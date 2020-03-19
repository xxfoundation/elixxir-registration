////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/testkeys"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	runFunc := func() int {
		code := m.Run()
		return code
	}

	os.Exit(runFunc())
}

// TestFunc: Gets permissioning server test key
func getTestKey() *rsa.PrivateKey {
	permKeyBytes, _ := utils.ReadFile(testkeys.GetCAKeyPath())

	testPermissioningKey, _ := rsa.LoadPrivateKeyFromPem(permKeyBytes)
	return testPermissioningKey
}

// Full test
func TestState_IsRoundNode(t *testing.T) {
	s, err := NewState(1)
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	s.currentRound = &RoundState{
		RoundInfo: &pb.RoundInfo{},
	}
	testString := "Test"

	// Test false case
	if s.IsRoundNode(testString) {
		t.Errorf("Expected node not to be round node!")
	}

	// Test true case
	s.currentRound.Topology = []string{testString}
	if !s.IsRoundNode(testString) {
		t.Errorf("Expected node to be round node!")
	}
}

// Full test
func TestState_GetCurrentRoundState(t *testing.T) {
	s, err := NewState(1)
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}

	// Test nil case
	if s.GetCurrentRoundState() != states.COMPLETED {
		t.Errorf("Expected nil round to return completed state! Got %+v",
			s.GetCurrentRoundState())
	}

	s.currentRound = &RoundState{
		RoundInfo: &pb.RoundInfo{
			State: uint32(states.FAILED),
		},
	}

	// Test happy path
	if s.GetCurrentRoundState() != states.FAILED {
		t.Errorf("Expected proper state return! Got %d", s.GetCurrentRoundState())
	}
}

// Happy path
func TestState_CreateNextRound(t *testing.T) {
	batchSize := uint32(1)
	topology := []string{"test", "test2"}

	s, err := NewState(batchSize)
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	s.PrivateKey = getTestKey()

	err = s.CreateNextRound(topology)
	if err != nil {
		t.Errorf("Unexpected error creating round: %+v", err)
	}

	// Check attributes
	if s.currentRound.GetID() != 0 {
		t.Errorf("Incorrect round ID")
	}
	if s.currentRound.GetUpdateID() != 1 {
		t.Errorf("Incorrect update ID!")
	}
	if s.currentRound.GetState() != uint32(states.PRECOMPUTING) {
		t.Errorf("Incorrect round state!")
	}
	if s.currentRound.GetBatchSize() != batchSize {
		t.Errorf("Incorrect round batch size!")
	}
	if len(s.currentRound.Topology) != len(topology) {
		t.Errorf("Incorrect round topology!")
	}

	// Check node statuses
	for _, status := range s.currentRound.nodeStatuses {
		if *status != uint32(states.PENDING) {
			t.Errorf("Incorrect node status!")
		}
	}

	// Check round signature
	if s.currentRound.RoundInfo.GetSignature() == nil ||
		len(s.currentRound.RoundInfo.GetNonce()) < 1 {
		t.Errorf("Incorrect round signature!")
	}

	// Check state data
	if _, err := s.roundData.GetRound(0); err != nil {
		t.Errorf("Incorrect round data: %+v", err)
	}

	// Check state updates
	if _, err := s.roundUpdates.GetUpdate(0); err != nil {
		t.Errorf("Incorrect round update data: %+v", err)
	}
}

// Full test
func TestState_UpdateNodeState(t *testing.T) {
	topology := []string{"test", "test2"}

	s, err := NewState(1)
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	s.PrivateKey = getTestKey()

	err = s.CreateNextRound(topology)
	if err != nil {
		t.Errorf("Unexpected error creating round: %+v", err)
	}

	// Test update without change in status
	err = s.UpdateNodeState(id.NewNodeFromBytes([]byte(topology[0])),
		states.PENDING)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if *s.currentRound.nodeStatuses[*id.NewNodeFromBytes([]byte(
		topology[0]))] != uint32(states.PENDING) {
		t.Errorf("Expected node status not to change!")
	}

	// Test updating node statuses
	err = s.UpdateNodeState(id.NewNodeFromBytes([]byte(topology[0])),
		states.PRECOMPUTING)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if *s.currentRound.nodeStatuses[*id.NewNodeFromBytes([]byte(
		topology[0]))] != uint32(states.PRECOMPUTING) {
		t.Errorf("Expected node status not to change!")
	}
	err = s.UpdateNodeState(id.NewNodeFromBytes([]byte(topology[1])),
		states.PRECOMPUTING)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if *s.currentRound.nodeStatuses[*id.NewNodeFromBytes([]byte(
		topology[1]))] != uint32(states.PRECOMPUTING) {
		t.Errorf("Expected node status not to change!")
	}

	// Test updating node statuses that trigger an incrementation
	err = s.UpdateNodeState(id.NewNodeFromBytes([]byte(topology[0])),
		states.STANDBY)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	err = s.UpdateNodeState(id.NewNodeFromBytes([]byte(topology[1])),
		states.STANDBY)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if s.currentRound.State != uint32(states.REALTIME) {
		t.Errorf("Expected round to increment! Got %s",
			states.Round(s.currentRound.State))
	}
}

// Happy path
func TestState_incrementRoundState(t *testing.T) {
	topology := []string{"test", "test2"}

	s, err := NewState(1)
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	s.PrivateKey = getTestKey()

	err = s.CreateNextRound(topology)
	if err != nil {
		t.Errorf("Unexpected error creating round: %+v", err)
	}

	// Test incrementing to each state
	err = s.incrementRoundState(states.PENDING)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.currentUpdate != 1 {
		t.Errorf("Unexpected round update occurred!")
	}
	err = s.incrementRoundState(states.PRECOMPUTING)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.currentUpdate != 1 {
		t.Errorf("Unexpected round update occurred!")
	}
	err = s.incrementRoundState(states.STANDBY)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.currentUpdate != 2 {
		t.Errorf("Round update failed to occur!")
	}
	err = s.incrementRoundState(states.REALTIME)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.currentUpdate != 2 {
		t.Errorf("Unexpected round update occurred!")
	}
	err = s.incrementRoundState(states.COMPLETED)
	if err != nil {
		t.Errorf("Unexpected error incrementing round state: %+v", err)
	}
	if s.currentUpdate != 3 {
		t.Errorf("Round update failed to occur!")
	}
}
