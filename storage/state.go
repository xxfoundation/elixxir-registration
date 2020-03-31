////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles network state tracking and control

package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"sync"
	"sync/atomic"
)

// Used for keeping track of NDF and Round state
type State struct {
	// State parameters ---
	PrivateKey *rsa.PrivateKey

	// Round state ---
	CurrentRound  *RoundState
	CurrentUpdate int // Round update counter
	RoundUpdates  *dataStructures.Updates
	RoundData     *dataStructures.Data
	Update        chan struct{} // For triggering updates to top level

	// NDF state ---
	ndfMux        sync.RWMutex
	partialNdf    *dataStructures.Ndf
	fullNdf       *dataStructures.Ndf
	PartialNdfMsg *pb.NDF
	FullNdfMsg    *pb.NDF
}

// Tracks the current global state of a round
type RoundState struct {
	// Tracks round information
	*pb.RoundInfo

	// Keeps track of the state of each node
	NodeStatuses map[id.Node]*uint32

	// Keeps track of the real state of the network
	// as described by the cumulative states of nodes
	// In other words, counts the number of nodes currently in each state
	NetworkStatus [states.NUM_STATES]*uint32
}

// Returns a new State object
func NewState() (*State, error) {
	state := &State{
		CurrentRound: &RoundState{
			RoundInfo: &pb.RoundInfo{
				Topology: make([]string, 0),        // Set this to avoid segfault
				State:    uint32(states.COMPLETED), // Set this to start rounds
			},
			NodeStatuses: make(map[id.Node]*uint32),
		},
		CurrentUpdate: 0,
		RoundUpdates:  dataStructures.NewUpdates(),
		RoundData:     dataStructures.NewData(),
		Update:        make(chan struct{}),
	}

	// Insert dummy update
	err := state.AddRoundUpdate(&pb.RoundInfo{})
	if err != nil {
		return nil, err
	}
	return state, nil
}

// Returns the full NDF
func (s *State) GetFullNdf() *dataStructures.Ndf {
	s.ndfMux.RLock()
	defer s.ndfMux.RUnlock()

	return s.fullNdf
}

// Returns the partial NDF
func (s *State) GetPartialNdf() *dataStructures.Ndf {
	s.ndfMux.RLock()
	defer s.ndfMux.RUnlock()

	return s.partialNdf
}

// Returns all updates after the given ID
func (s *State) GetUpdates(id int) ([]*pb.RoundInfo, error) {
	return s.RoundUpdates.GetUpdates(id)
}

// Returns true if given node ID is participating in the current round
func (s *State) IsRoundNode(id string) bool {
	for _, nodeId := range s.CurrentRound.Topology {
		if nodeId == id {
			return true
		}
	}
	return false
}

// Returns the state of the current round
func (s *State) GetCurrentRoundState() states.Round {
	return states.Round(s.CurrentRound.State)
}

// Makes a copy of the round before inserting into RoundUpdates
func (s *State) AddRoundUpdate(round *pb.RoundInfo) error {
	roundCopy := &pb.RoundInfo{
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
	jww.DEBUG.Printf("Round state updated to %s",
		states.Round(roundCopy.State))

	return s.RoundUpdates.AddRound(roundCopy)
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
	fullNdfMsg := &pb.NDF{}
	fullNdfMsg.Ndf, err = s.GetFullNdf().Get().Marshal()
	if err != nil {
		return
	}
	partialNdfMsg := &pb.NDF{}
	partialNdfMsg.Ndf, err = s.GetPartialNdf().Get().Marshal()
	if err != nil {
		return
	}

	// Sign NDF comms messages
	err = signature.Sign(fullNdfMsg, s.PrivateKey)
	if err != nil {
		return
	}
	err = signature.Sign(partialNdfMsg, s.PrivateKey)
	if err != nil {
		return
	}

	// Assign NDF comms messages
	s.ndfMux.Lock()
	s.FullNdfMsg = fullNdfMsg
	s.PartialNdfMsg = partialNdfMsg
	s.ndfMux.Unlock()
	return nil
}

// Updates the state of the given node with the new state provided
func (s *State) UpdateNodeState(id *id.Node, newState states.Round) {
	// Attempt to update node state atomically
	// If an update occurred, continue, else nothing will happen
	old := atomic.SwapUint32(s.CurrentRound.NodeStatuses[*id], uint32(newState))
	if old != uint32(newState) {
		// Node state was updated, increment state counter
		atomic.AddUint32(s.CurrentRound.NetworkStatus[newState], 1)

		// Cue an update
		s.Update <- struct{}{}
	}
}
