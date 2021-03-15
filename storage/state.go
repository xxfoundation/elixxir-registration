////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles network state tracking and control

package storage

import (
	"github.com/golang-collections/collections/set"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
	"strconv"
	"strings"
	"sync"
	"time"
)

const updateBufferLength = 10000

// NetworkState structure used for keeping track of NDF and Round state.
type NetworkState struct {
	// NetworkState parameters
	privateKey *rsa.PrivateKey

	// Round state
	rounds       *round.StateMap
	roundUpdates *dataStructures.Updates
	roundData    *dataStructures.Data
	update       chan node.UpdateNotification // For triggering updates to top level

	// Node NetworkState
	nodes     *node.StateMap
	updateMux sync.Mutex

	// List of states of Nodes to be disabled
	disabledNodesStates *disabledNodes

	// NDF state

	unprunedNdf *ndf.NetworkDefinition

	pruneListMux sync.RWMutex
	pruneList  map[id.ID]interface{}
	partialNdf *dataStructures.Ndf
	fullNdf    *dataStructures.Ndf

	// Address space size
	addressSpaceSize uint32

	ndfOutputPath string
}

// NewState returns a new NetworkState object.
func NewState(pk *rsa.PrivateKey, addressSpaceSize uint32, ndfOutputPath string) (*NetworkState, error) {
	fullNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		return nil, err
	}
	partialNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		return nil, err
	}

	state := &NetworkState{
		rounds:           round.NewStateMap(),
		roundUpdates:     dataStructures.NewUpdates(),
		update:           make(chan node.UpdateNotification, updateBufferLength),
		nodes:            node.NewStateMap(),
		unprunedNdf: 	  &ndf.NetworkDefinition{},
		fullNdf:          fullNdf,
		partialNdf:       partialNdf,
		privateKey:       pk,
		addressSpaceSize: addressSpaceSize,
		pruneList: 		  make(map[id.ID]interface{}),
		ndfOutputPath: 	  ndfOutputPath,
	}

	// Obtain round & update Id from Storage
	// Ignore not found in Storage errors, zero-value will be handled below
	updateId, err := state.GetUpdateID()
	if err != nil &&
		!strings.Contains(err.Error(), gorm.ErrRecordNotFound.Error()) &&
		!strings.Contains(err.Error(), "Unable to locate state for key") {
		return nil, err
	}
	roundId, err := state.GetRoundID()
	if err != nil &&
		!strings.Contains(err.Error(), gorm.ErrRecordNotFound.Error()) &&
		!strings.Contains(err.Error(), "Unable to locate state for key") {
		return nil, err
	}

	// Updates are handled in the uint space, as a result, the designator for
	// update 0 also designates that no updates are known by the server. To
	// avoid this collision, permissioning will skip this update as well.
	if updateId == 0 {
		// Set update Id to start at 0
		err = state.setId(UpdateIdKey, 0)
		if err != nil {
			return nil, err
		}
		// Then insert a dummy and increment to 1
		err = state.AddRoundUpdate(&pb.RoundInfo{})
		if err != nil {
			return nil, err
		}
	}
	if roundId == 0 {
		// Set round Id to start at 1
		err = state.setId(RoundIdKey, 1)
		if err != nil {
			return nil, err
		}
	}

	return state, nil
}

func (s *NetworkState) SetPrunedNodes(ids []*id.ID) {
	s.pruneListMux.Lock()
	defer s.pruneListMux.Unlock()

	s.pruneList = make(map[id.ID]interface{})

	for _, i := range ids{
		s.pruneList[*i]=nil
	}
}

func (s *NetworkState) SetPrunedNode(id *id.ID) {
	s.pruneListMux.Lock()
	defer s.pruneListMux.Unlock()

	s.pruneList[*id]=nil
}

func (s *NetworkState) IsPruned(node *id.ID)bool {
	s.pruneListMux.RLock()
	defer s.pruneListMux.RUnlock()

	_, exists := s.pruneList[*node]
	return exists
}

func (s *NetworkState) GetUnprunedNdf() *ndf.NetworkDefinition {
	return s.unprunedNdf
}

// GetFullNdf returns the full NDF.
func (s *NetworkState) GetFullNdf() *dataStructures.Ndf {
	return s.fullNdf
}

// GetPartialNdf returns the partial NDF.
func (s *NetworkState) GetPartialNdf() *dataStructures.Ndf {
	return s.partialNdf
}

// GetUpdates returns all of the updates after the given ID.
func (s *NetworkState) GetUpdates(id int) ([]*pb.RoundInfo, error) {
	return s.roundUpdates.GetUpdates(id), nil
}

// AddRoundUpdate creates a copy of the round before inserting it into
// roundUpdates.
func (s *NetworkState) AddRoundUpdate(r *pb.RoundInfo) error {
	s.updateMux.Lock()
	defer s.updateMux.Unlock()

	roundCopy := round.CopyRoundInfo(r)
	updateID, err := s.IncrementUpdateID()
	if err != nil {
		return err
	}

	roundCopy.UpdateID = updateID

	err = signature.Sign(roundCopy, s.privateKey)
	if err != nil {
		return errors.WithMessagef(err, "Could not add round update %v "+
			"for round %v due to failed signature", roundCopy.UpdateID, roundCopy.ID)
	}

	jww.INFO.Printf("Round %v state updated to %s", r.ID,
		states.Round(roundCopy.State))

	jww.TRACE.Printf("Round Info: %+v", roundCopy)

	return s.roundUpdates.AddRound(roundCopy)
}

// UpdateNdf updates internal NDF structures with the specified new NDF.
func (s *NetworkState) UpdateNdf(newNdf *ndf.NetworkDefinition) (err error) {

	ndfMarshabled, _ := newNdf.Marshal()
	s.unprunedNdf, _ = ndf.Unmarshal(ndfMarshabled)

	s.pruneListMux.RLock()

	//prune the NDF
	for i := 0; i < len(newNdf.Nodes);i++ {
		nid, _ := id.Unmarshal(newNdf.Nodes[i].ID)
		if _, exists := s.pruneList[*nid]; exists{
			newNdf.Nodes = append(newNdf.Nodes[:i], newNdf.Nodes[i+1:]...)
			newNdf.Gateways = append(newNdf.Gateways[:i], newNdf.Gateways[i+1:]...)
			i--
		}
	}

	s.pruneListMux.RUnlock()


	// Build NDF comms messages
	fullNdfMsg := &pb.NDF{}
	fullNdfMsg.Ndf, err = newNdf.Marshal()
	if err != nil {
		return
	}
	partialNdfMsg := &pb.NDF{}
	partialNdfMsg.Ndf, err = newNdf.StripNdf().Marshal()
	if err != nil {
		return
	}

	// Sign NDF comms messages
	err = signature.Sign(fullNdfMsg, s.privateKey)
	if err != nil {
		return
	}
	err = signature.Sign(partialNdfMsg, s.privateKey)
	if err != nil {
		return
	}

	// Assign NDF comms messages
	err = s.fullNdf.Update(fullNdfMsg)
	if err != nil {
		return err
	}

	err = s.partialNdf.Update(partialNdfMsg)
	if err != nil {
		return err
	}

	err = outputToJSON(newNdf, s.ndfOutputPath)
	if err != nil {
		jww.ERROR.Printf("unable to output NDF JSON file: %+v", err)
	}

	return nil
}

// GetPrivateKey returns the server's private key.
func (s *NetworkState) GetPrivateKey() *rsa.PrivateKey {
	return s.privateKey
}

// GetRoundMap returns the map of rounds.
func (s *NetworkState) GetRoundMap() *round.StateMap {
	return s.rounds
}

// GetNodeMap returns the map of nodes.
func (s *NetworkState) GetNodeMap() *node.StateMap {
	return s.nodes
}

// GetAddressSpaceSize returns the address space size
func (s *NetworkState) GetAddressSpaceSize() uint32 {
	return s.addressSpaceSize
}

// NodeUpdateNotification sends a notification to the control thread of an
// update to a nodes state.
func (s *NetworkState) SendUpdateNotification(nun node.UpdateNotification) error {
	select {
	case s.update <- nun:
		return nil
	default:
		return errors.New("Could not send update notification")
	}
}

// GetNodeUpdateChannel returns a channel to receive node updates on.
func (s *NetworkState) GetNodeUpdateChannel() <-chan node.UpdateNotification {
	return s.update
}

// Helper to increment the RoundId or UpdateId depending on the given key
// FIXME: Get and set should be coupled to avoid race conditions
func (s *NetworkState) increment(key string) (uint64, error) {
	oldIdStr, err := PermissioningDb.GetStateValue(key)
	if err != nil {
		return 0, errors.Errorf("Unable to obtain current %s: %+v", key, err)
	}

	oldId, err := strconv.ParseUint(oldIdStr, 10, 64)
	if err != nil {
		return 0, errors.Errorf("Unable to parse current %s: %+v", key, err)
	}

	return oldId, s.setId(key, oldId+1)
}

// Helper to set the roundId or updateId value
func (s *NetworkState) setId(key string, newVal uint64) error {
	err := PermissioningDb.UpsertState(&State{
		Key:   key,
		Value: strconv.FormatUint(newVal, 10),
	})
	if err != nil {
		return errors.Errorf("Unable to update current round ID: %+v", err)
	}
	return nil
}

// Helper to return the RoundId or UpdateId depending on the given key
func (s *NetworkState) get(key string) (uint64, error) {
	roundIdStr, err := PermissioningDb.GetStateValue(key)
	if err != nil {
		return 0, errors.Errorf("Unable to obtain current %s: %+v", key, err)
	}

	roundId, err := strconv.ParseUint(roundIdStr, 10, 64)
	if err != nil {
		return 0, errors.Errorf("Unable to parse current %s: %+v", key, err)
	}
	return roundId, nil
}

// IncrementRoundID increments the round ID
func (s *NetworkState) IncrementRoundID() (id.Round, error) {
	roundId, err := s.increment(RoundIdKey)
	return id.Round(roundId), err
}

// IncrementUpdateID increments the update ID
func (s *NetworkState) IncrementUpdateID() (uint64, error) {
	return s.increment(UpdateIdKey)
}

// GetRoundID returns the round ID
func (s *NetworkState) GetRoundID() (id.Round, error) {
	roundId, err := s.get(RoundIdKey)
	return id.Round(roundId), err
}

// GetRoundID returns the update ID
func (s *NetworkState) GetUpdateID() (uint64, error) {
	return s.get(UpdateIdKey)
}

// CreateDisabledNodes generates and sets a disabledNodes object that will track
// disabled Nodes list.
func (s *NetworkState) CreateDisabledNodes(path string, interval time.Duration) error {
	var err error
	s.disabledNodesStates, err = generateDisabledNodes(path, interval, s)
	return err
}

// StartPollDisabledNodes starts the loop that polls for updates
func (s *NetworkState) StartPollDisabledNodes(quitChan chan struct{}) {
	s.disabledNodesStates.pollDisabledNodes(s, quitChan)
}

// GetDisabledNodesSet returns the set of states of disabled nodes.
func (s *NetworkState) GetDisabledNodesSet() *set.Set {
	if s.disabledNodesStates != nil {
		return s.disabledNodesStates.getDisabledNodes()
	}

	return nil
}


// outputNodeTopologyToJSON encodes the NodeTopology structure to JSON and
// outputs it to the specified file path. An error is returned if the JSON
// marshaling fails or if the JSON file cannot be created.
func outputToJSON(ndfData *ndf.NetworkDefinition, filePath string) error {
	// Generate JSON from structure
	data, err := ndfData.Marshal()
	if err != nil {
		return err
	}
	// Write JSON to file
	return utils.WriteFile(filePath, data, utils.FilePerms, utils.DirPerms)
}
