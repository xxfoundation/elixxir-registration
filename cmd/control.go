////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles control layer above the network state

package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"time"
)

// Control thread for advancement of network state
func (m *RegistrationImpl) StateControl() {
	s := m.State
	for range s.Update {

		// Check whether the round state is ready to increment
		nextState := states.Round(s.CurrentRound.State + 1)
		numNodesInRound := uint32(len(s.CurrentRound.Topology))
		if *s.CurrentRound.NetworkStatus[nextState] == numNodesInRound {
			// Increment the round state
			err := m.incrementRoundState(nextState)
			if err != nil {
				// TODO: Error handling
				jww.FATAL.Panicf("Unable to create next round: %+v", err)
			}
		}

		// Handle completion of a round
		if s.GetCurrentRoundState() == states.COMPLETED {
			// Create the new round
			err := m.newRound(s.CurrentRound.GetTopology(), m.params.batchSize)
			if err != nil {
				// TODO: Error handling
				jww.FATAL.Panicf("Unable to create next round: %+v", err)
			}
		}
	}
}

// Initiate the next round with a selection of nodes
func (m *RegistrationImpl) createNextRound() error {
	// Build a topology (currently consisting of all nodes in network)
	var topology []string
	for _, node := range m.State.GetPartialNdf().Get().Nodes {
		topology = append(topology, node.GetNodeId().String())
	}

	// Progress to the next round
	return m.newRound(topology, m.params.batchSize)
}

// Increments the state of the current round if needed
func (m *RegistrationImpl) incrementRoundState(state states.Round) error {
	s := m.State

	// Handle state transitions
	switch state {
	case states.STANDBY:
		s.CurrentRound.State = uint32(states.REALTIME)
		// Handle timestamp edge case with realtime
		s.CurrentRound.Timestamps[states.REALTIME] = uint64(time.Now().Add(2 * time.Second).Unix())
	case states.COMPLETED:
		s.CurrentRound.State = uint32(states.COMPLETED)
		s.CurrentRound.Timestamps[states.COMPLETED] = uint64(time.Now().Unix())
	default:
		return nil
	}
	// Update current round state
	s.CurrentUpdate += 1
	s.CurrentRound.UpdateID = uint64(s.CurrentUpdate)

	// Sign the new round object
	err := signature.Sign(s.CurrentRound.RoundInfo, s.PrivateKey)
	if err != nil {
		return err
	}

	// Insert update into the state tracker
	return s.AddRoundUpdate(s.CurrentRound.RoundInfo)
}

// Builds and inserts the next RoundInfo object into the internal state
func (m *RegistrationImpl) newRound(topology []string, batchSize uint32) error {
	s := m.State

	// Build the new current round object
	s.CurrentUpdate += 1
	s.CurrentRound.RoundInfo = &pb.RoundInfo{
		ID:         uint64(s.RoundData.GetLastRoundID() + 1),
		UpdateID:   uint64(s.CurrentUpdate),
		State:      uint32(states.PRECOMPUTING),
		BatchSize:  batchSize,
		Topology:   topology,
		Timestamps: make([]uint64, states.NUM_STATES),
	}
	s.CurrentRound.Timestamps[states.PRECOMPUTING] = uint64(time.Now().Unix())
	jww.DEBUG.Printf("Initializing round %d...", s.CurrentRound.ID)

	// Initialize network status
	for i := range s.CurrentRound.NetworkStatus {
		val := uint32(0)
		s.CurrentRound.NetworkStatus[i] = &val
	}

	// Initialize node states based on given topology
	for _, nodeId := range topology {
		newState := uint32(states.PENDING)
		s.CurrentRound.NodeStatuses[*id.NewNodeFromString(nodeId)] = &newState
	}

	// Sign the new round object
	err := signature.Sign(s.CurrentRound.RoundInfo, s.PrivateKey)
	if err != nil {
		return err
	}

	// Insert the new round object into the state tracker
	err = s.RoundData.UpsertRound(s.CurrentRound.RoundInfo)
	if err != nil {
		return err
	}
	return s.AddRoundUpdate(s.CurrentRound.RoundInfo)
}

// Attempt to update the internal state after a node polling operation
func (m *RegistrationImpl) updateState(id *id.Node, activity *current.Activity) error {
	// Convert node activity to round state
	roundState, err := activity.ConvertToRoundState()
	if err != nil {
		return err
	}

	// Update node state
	m.State.UpdateNodeState(id, roundState)
	return nil
}
