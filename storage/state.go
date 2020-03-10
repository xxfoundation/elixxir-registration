////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles network state tracking and control

package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"sync"
	"sync/atomic"
	"time"
)

// Used for keeping track of NDF and Round state
type State struct {
	NumNodes   uint32
	PrivateKey *rsa.PrivateKey
	batchSize  uint32

	// Round state
	currentRound *RoundState
	roundUpdates *dataStructures.Updates
	roundData    *dataStructures.Data

	// NDF state
	partialNdf    *dataStructures.Ndf
	fullNdf       *dataStructures.Ndf
	PartialNdfMsg *pb.NDF
	FullNdfMsg    *pb.NDF
}

// Tracks the current global state of a round
type RoundState struct {
	// Tracks round information
	*pb.RoundInfo
	// Mutex associated with roundInfo
	mux sync.RWMutex

	// Keeps track of the state of each node
	nodeStatuses map[*id.Node]*uint32

	// Keeps track of the real state of the network
	// as described by the cumulative states of nodes
	// In other words, counts the number of nodes currently in each state
	networkStatus [states.NUM_STATES]*uint32
}

// Returns a new State object
func NewState(numNodes, batchSize uint32) *State {
	return &State{
		NumNodes:     numNodes,
		batchSize:    batchSize,
		roundUpdates: dataStructures.NewUpdates(),
		roundData:    dataStructures.NewData(),
	}
}

// Builds and inserts the next RoundInfo object into the internal state
func (s *State) CreateNextRoundInfo(topology []string) error {
	s.currentRound.mux.Lock()
	defer s.currentRound.mux.Unlock()

	// Build the new current round object
	s.currentRound = &RoundState{
		RoundInfo: &mixmessages.RoundInfo{
			ID:         uint64(s.roundData.GetLastRoundID() + 1),
			UpdateID:   uint64(s.roundUpdates.GetLastUpdateID() + 1),
			State:      uint32(states.PENDING),
			BatchSize:  s.batchSize,
			Topology:   topology,
			Timestamps: []uint64{uint64(time.Now().Unix())},
		},
	}
	jww.DEBUG.Printf("Initializing round %d...", s.currentRound.ID)

	// Initialize node states based on given topology
	for _, nodeId := range topology {
		newState := uint32(states.PENDING)
		s.currentRound.nodeStatuses[id.NewNodeFromBytes([]byte(
			nodeId))] = &newState
	}

	// Sign the new round object
	err := signature.Sign(s.currentRound.RoundInfo, s.PrivateKey)
	if err != nil {
		return err
	}

	// Insert the new round object into the state tracker
	err = s.roundData.UpsertRound(s.currentRound.RoundInfo)
	if err != nil {
		return err
	}
	return s.addRoundUpdate(s.currentRound.RoundInfo)
}

// Makes a copy of the round before inserting into roundUpdates
func (s *State) addRoundUpdate(round *pb.RoundInfo) error {
	roundCopy := &mixmessages.RoundInfo{
		ID:         round.GetID(),
		UpdateID:   round.GetUpdateID(),
		State:      round.GetState(),
		BatchSize:  round.GetBatchSize(),
		Topology:   round.GetTopology(),
		Timestamps: round.GetTimestamps(),
		Signature: &pb.RSASignature{
			Nonce:     round.GetNonce(),
			Signature: round.GetSig(),
		},
	}

	return s.roundUpdates.AddRound(roundCopy)
}

// Given a full NDF, updates internal NDF structures
func (s *State) UpdateNdf(newNdf *ndf.NetworkDefinition) (err error) {
	s.fullNdf, err = dataStructures.NewNdf(newNdf)
	if err != nil {
		return
	}
	s.partialNdf, err = dataStructures.NewNdf(newNdf.StripNdf())
	if err != nil {
		return
	}

	// Build NDF comms messages
	s.FullNdfMsg.Ndf, err = s.GetFullNdf().Get().Marshal()
	if err != nil {
		return
	}
	s.PartialNdfMsg.Ndf, err = s.GetPartialNdf().Get().Marshal()
	if err != nil {
		return
	}

	// Sign NDF comms messages
	err = signature.Sign(s.FullNdfMsg, s.PrivateKey)
	if err != nil {
		return
	}
	return signature.Sign(s.PartialNdfMsg, s.PrivateKey)
}

// Increments the state of the current round if needed
func (s *State) incrementRoundState(state states.Round) error {
	s.currentRound.mux.Lock()
	defer s.currentRound.mux.Unlock()

	// Handle state transitions
	switch state {
	case states.PENDING:
		s.currentRound.State = uint32(states.PRECOMPUTING)
		s.currentRound.Timestamps = append(s.currentRound.Timestamps,
			uint64(time.Now().Unix()))
	case states.STANDBY:
		s.currentRound.State = uint32(states.REALTIME)
		// Handle timestamp edge case with realtime
		s.currentRound.Timestamps = append(s.currentRound.Timestamps,
			uint64(time.Now().Add(2*time.Second).Unix()))
	case states.COMPLETED:
		s.currentRound.State = uint32(states.COMPLETED)
		s.currentRound.Timestamps = append(s.currentRound.Timestamps,
			uint64(time.Now().Unix()))
	default:
		return nil
	}
	// Update current round state
	s.currentRound.UpdateID += 1

	// Sign the new round object
	err := signature.Sign(s.currentRound.RoundInfo, s.PrivateKey)
	if err != nil {
		return err
	}
	jww.DEBUG.Printf("Round state incremented to %s",
		states.Round(s.currentRound.State))

	// Insert update into the state tracker
	return s.addRoundUpdate(s.currentRound.RoundInfo)
}

// Returns the full NDF
func (s *State) GetFullNdf() *dataStructures.Ndf {
	return s.fullNdf
}

// Returns the partial NDF
func (s *State) GetPartialNdf() *dataStructures.Ndf {
	return s.partialNdf
}

// Returns all updates after the given ID
func (s *State) GetUpdates(id int) ([]*pb.RoundInfo, error) {
	return s.roundUpdates.GetUpdates(id)
}

// Returns true if given node ID is participating in the current round
func (s *State) IsRoundNode(id string) bool {
	s.currentRound.mux.RLock()
	defer s.currentRound.mux.RUnlock()

	for _, nodeId := range s.currentRound.Topology {
		if nodeId == id {
			return true
		}
	}
	return false
}

// Returns the state of the current round
func (s *State) GetCurrentRoundState() states.Round {
	s.currentRound.mux.RLock()
	defer s.currentRound.mux.RUnlock()

	// If no round has been started, set to COMPLETE
	if s.currentRound == nil {
		return states.COMPLETED
	}
	return states.Round(s.currentRound.State)
}

// Updates the state of the given node with the new state provided
func (s *State) UpdateNodeState(id *id.Node, newState states.Round) error {
	// Attempt to update node state atomically
	// If an update occurred, continue, else nothing will happen
	if old := atomic.SwapUint32(
		s.currentRound.nodeStatuses[id], uint32(newState)); old != uint32(newState) {

		// Node state was updated, increment state counter
		result := atomic.AddUint32(s.currentRound.networkStatus[newState], 1)

		// Check whether the round state is ready to increment
		if result == s.NumNodes {
			return s.incrementRoundState(newState)
		}
	}

	// If node state hasn't changed, do nothing
	return nil
}