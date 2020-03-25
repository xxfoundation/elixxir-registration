////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"testing"
)

// Happy path
func Test_createNextRound(t *testing.T) {
	batchSize := uint32(1)
	topology := []string{"test", "test2"}

	s, err := storage.NewState()
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	impl := &RegistrationImpl{
		State: s,
	}
	s.PrivateKey = getTestKey()

	err = impl.newRound(topology, batchSize)
	if err != nil {
		t.Errorf("Unexpected error creating round: %+v", err)
	}

	// Check attributes
	if s.CurrentRound.GetID() != 0 {
		t.Errorf("Incorrect round ID")
	}
	if s.CurrentRound.GetUpdateID() != 1 {
		t.Errorf("Incorrect update ID!")
	}
	if s.CurrentRound.GetState() != uint32(states.PRECOMPUTING) {
		t.Errorf("Incorrect round state!")
	}
	if s.CurrentRound.GetBatchSize() != batchSize {
		t.Errorf("Incorrect round batch size!")
	}
	if len(s.CurrentRound.Topology) != len(topology) {
		t.Errorf("Incorrect round topology!")
	}

	// Check node statuses
	for _, status := range s.CurrentRound.NodeStatuses {
		if *status != uint32(states.PENDING) {
			t.Errorf("Incorrect node status!")
		}
	}

	// Check round signature
	if s.CurrentRound.RoundInfo.GetSignature() == nil ||
		len(s.CurrentRound.RoundInfo.GetNonce()) < 1 {
		t.Errorf("Incorrect round signature!")
	}

	// Check state data
	if _, err := s.RoundData.GetRound(0); err != nil {
		t.Errorf("Incorrect round data: %+v", err)
	}

	// Check state updates
	if _, err := s.RoundUpdates.GetUpdate(0); err != nil {
		t.Errorf("Incorrect round update data: %+v", err)
	}
}

// Full test
func Test_updateState(t *testing.T) {
	topology := []string{"test", "test2"}

	s, err := storage.NewState()
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}

	impl := &RegistrationImpl{
		State: s,
		params: &Params{
			batchSize: 1,
		},
	}
	s.PrivateKey = getTestKey()

	go impl.StateControl()

	err = impl.newRound(topology, impl.params.batchSize)
	if err != nil {
		t.Errorf("Unexpected error creating round: %+v", err)
	}

	// Test update without change in status
	state := current.WAITING
	err = impl.updateState(id.NewNodeFromBytes([]byte(topology[0])), &state)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if *s.CurrentRound.NodeStatuses[*id.NewNodeFromBytes([]byte(
		topology[0]))] != uint32(states.PENDING) {
		t.Errorf("Expected node status not to change!")
	}

	// Test updating node statuses
	state = current.PRECOMPUTING
	err = impl.updateState(id.NewNodeFromBytes([]byte(topology[0])), &state)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if *s.CurrentRound.NodeStatuses[*id.NewNodeFromBytes([]byte(
		topology[0]))] != uint32(states.PRECOMPUTING) {
		t.Errorf("Expected node status not to change!")
	}
	err = impl.updateState(id.NewNodeFromBytes([]byte(topology[1])), &state)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if *s.CurrentRound.NodeStatuses[*id.NewNodeFromBytes([]byte(
		topology[1]))] != uint32(states.PRECOMPUTING) {
		t.Errorf("Expected node status not to change!")
	}

	// Test updating node statuses that trigger an incrementation
	state = current.STANDBY
	err = impl.updateState(id.NewNodeFromBytes([]byte(topology[0])),
		&state)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	err = impl.updateState(id.NewNodeFromBytes([]byte(topology[1])),
		&state)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if s.CurrentRound.State != uint32(states.REALTIME) {
		t.Errorf("Expected round to increment! Got %s",
			states.Round(s.CurrentRound.State))
	}
}

// Happy path
func Test_incrementRoundState(t *testing.T) {
	topology := []string{"test", "test2"}

	s, err := storage.NewState()
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}

	impl := &RegistrationImpl{
		State: s,
		params: &Params{
			batchSize: 1,
		},
	}
	s.PrivateKey = getTestKey()

	err = impl.newRound(topology, impl.params.batchSize)
	if err != nil {
		t.Errorf("Unexpected error creating round: %+v", err)
	}

	// Test incrementing to each state
	err = impl.incrementRoundState(states.PENDING)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.CurrentUpdate != 1 {
		t.Errorf("Unexpected round update occurred!")
	}
	err = impl.incrementRoundState(states.PRECOMPUTING)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.CurrentUpdate != 1 {
		t.Errorf("Unexpected round update occurred!")
	}
	err = impl.incrementRoundState(states.STANDBY)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.CurrentUpdate != 2 {
		t.Errorf("Round update failed to occur!")
	}
	err = impl.incrementRoundState(states.REALTIME)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.CurrentUpdate != 2 {
		t.Errorf("Unexpected round update occurred!")
	}
	err = impl.incrementRoundState(states.COMPLETED)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.CurrentUpdate != 3 {
		t.Errorf("Round update failed to occur!")
	}
}
