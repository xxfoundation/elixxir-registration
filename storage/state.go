////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles network state tracking and control

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"sync"
)

const updateBufferLength = 1000

// NetworkState structure used for keeping track of NDF and Round state.
type NetworkState struct {
	// NetworkState parameters
	privateKey *rsa.PrivateKey

	// The ID of the current round
	roundID *stateID

	// Round state
	rounds          *round.StateMap
	roundUpdates    *dataStructures.Updates
	roundUpdateID   *stateID
	roundUpdateLock sync.Mutex
	roundData       *dataStructures.Data
	update          chan node.UpdateNotification // For triggering updates to top level

	// Node NetworkState
	nodes *node.StateMap

	// NDF state
	partialNdf *dataStructures.Ndf
	fullNdf    *dataStructures.Ndf
}

// NewState returns a new NetworkState object.
func NewState(pk *rsa.PrivateKey, roundIdPath, updateIdPath string) (*NetworkState, error) {
	fullNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		return nil, err
	}
	partialNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		return nil, err
	}

	// Create round ID
	roundID, err := loadOrCreateStateID(roundIdPath, 1)
	if err != nil {
		return nil, errors.Errorf("Failed to load round ID from path: %+v", err)
	}

	// Create increment ID
	updateRoundID, err := loadOrCreateStateID(updateIdPath, 0)
	if err != nil {
		return nil, errors.Errorf("Failed to load update ID from path: %+v", err)
	}

	state := &NetworkState{
		roundID:       roundID,
		rounds:        round.NewStateMap(),
		roundUpdates:  dataStructures.NewUpdates(),
		update:        make(chan node.UpdateNotification, updateBufferLength),
		nodes:         node.NewStateMap(),
		fullNdf:       fullNdf,
		partialNdf:    partialNdf,
		privateKey:    pk,
		roundUpdateID: updateRoundID,
	}

	// Updates are handled in the uint space, as a result, the designator for
	// update 0 also designates that no updates are known by the server. To
	// avoid this collision, permissioning will skip this update as well.
	if updateRoundID.get() == 0 {
		// Insert dummy update
		err = state.AddRoundUpdate(&pb.RoundInfo{})
		if err != nil {
			return nil, err
		}
	}

	return state, nil
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
	s.roundUpdateLock.Lock()
	defer s.roundUpdateLock.Unlock()

	roundCopy := round.CopyRoundInfo(r)

	updateID, err := s.roundUpdateID.increment()
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
	return s.partialNdf.Update(partialNdfMsg)
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

// IncrementRoundID increments the round ID in a thread safe manner. If an error
// occurs while updating the ID file, then it is returned.
func (s *NetworkState) IncrementRoundID() (id.Round, error) {
	roundID, err := s.roundID.increment()
	return id.Round(roundID), err
}

// GetRoundID returns the round ID in a thread safe manner.
func (s *NetworkState) GetRoundID() id.Round {
	return id.Round(s.roundID.get())
}
