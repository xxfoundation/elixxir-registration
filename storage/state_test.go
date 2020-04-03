////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
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

// Full test
func TestState_IsRoundNode(t *testing.T) {
	s, err := NewState()
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	s.CurrentRound = &RoundState{
		RoundInfo: &pb.RoundInfo{},
	}
	testString := "Test"

	// Test false case
	if s.IsRoundNode(testString) {
		t.Errorf("Expected node not to be round node!")
	}

	// Test true case
	s.CurrentRound.Topology = []string{testString}
	if !s.IsRoundNode(testString) {
		t.Errorf("Expected node to be round node!")
	}
}

// Full test
func TestState_GetCurrentRoundState(t *testing.T) {
	s, err := NewState()
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}

	// Test nil case
	if s.GetCurrentRoundState() != states.COMPLETED {
		t.Errorf("Expected nil round to return completed state! Got %+v",
			s.GetCurrentRoundState())
	}

	s.CurrentRound = &RoundState{
		RoundInfo: &pb.RoundInfo{
			State: uint32(states.FAILED),
		},
	}

	// Test happy path
	if s.GetCurrentRoundState() != states.FAILED {
		t.Errorf("Expected proper state return! Got %d", s.GetCurrentRoundState())
	}
}
