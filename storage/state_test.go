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

// Gets permissioning server test key
func getTestKey() *rsa.PrivateKey {
	permKeyBytes, _ := utils.ReadFile(testkeys.GetCAKeyPath())

	testPermissioningKey, _ := rsa.LoadPrivateKeyFromPem(permKeyBytes)
	return testPermissioningKey

}

// Full test
func TestState_IsRoundNode(t *testing.T) {
	s := NewState(1, 1)
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
	s := NewState(1, 1)

	// Test nil case
	if s.GetCurrentRoundState() != states.COMPLETED {
		t.Errorf("Expected nil round to return completed state!")
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

	s := NewState(1, batchSize)
	s.PrivateKey = getTestKey()

	err := s.CreateNextRound(topology)
	if err != nil {
		t.Errorf("Expected no error for CreateNextRound: %+v", err)
	}

	// Check attributes
	if s.currentRound.GetID() != 0 {
		t.Errorf("Incorrect round ID")
	}
	if s.currentRound.GetUpdateID() != 0 {
		t.Errorf("Incorrect update ID!")
	}
	if s.currentRound.GetState() != uint32(states.PENDING) {
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
	batchSize := uint32(1)
	topology := []string{"test", "test2"}

	s := NewState(1, batchSize)
	s.PrivateKey = getTestKey()

	err := s.CreateNextRound(topology)
	if err != nil {
		t.Errorf("Expected no error for CreateNextRound: %+v", err)
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

	// Test update with a change in status
	err = s.UpdateNodeState(id.NewNodeFromBytes([]byte(topology[0])),
		states.PRECOMPUTING)
	if err != nil {
		t.Errorf("Unexpected error updating node state: %+v", err)
	}
	if *s.currentRound.nodeStatuses[*id.NewNodeFromBytes([]byte(
		topology[0]))] != uint32(states.PRECOMPUTING) {
		t.Errorf("Expected node status not to change!")
	}
}
