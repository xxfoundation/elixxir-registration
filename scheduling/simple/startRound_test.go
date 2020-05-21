////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"crypto/rand"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"strconv"
	"testing"
)

// Happy path
func TestStartRound(t *testing.T) {
	// Build params for scheduling
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build node list
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "")
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState

	}

	// Build pool
	testPool := &waitingPoll{
		pool:     nodeList,
		position: int(testParams.TeamSize),
	}

	roundID := NewRoundID(0)

	testProtoRound, err := createRound(testParams, testPool, roundID.Get(), testState)
	if err != nil {
		t.Errorf("Happy path of createRound failed: %v", err)
	}

	errorChan := make(chan error, 1)

	err = startRound(testProtoRound, testState, errorChan)
	if err != nil {
		t.Errorf("Received error from startRound(): %v", err)
	}

	if testState.GetRoundMap().GetRound(0).GetRoundState() != states.PRECOMPUTING {
		t.Errorf("In unexpected state after round creation: %v",
			testState.GetRoundMap().GetRound(0).GetRoundState())
	}
}

// Error path
func TestStartRound_BadState(t *testing.T) {
	// Build params for scheduling
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build node list
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "")
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState

	}

	// Build pool
	testPool := &waitingPoll{
		pool:     nodeList,
		position: int(testParams.TeamSize),
	}

	roundID := NewRoundID(0)

	// Manually set the state of the round
	badState := round.NewState_Testing(roundID.Get(), states.COMPLETED, t)
	testState.GetRoundMap().AddRound_Testing(badState, t)

	testProtoRound, err := createRound(testParams, testPool, roundID.Get(), testState)
	if err != nil {
		t.Errorf("Happy path of createRound failed: %v", err)
	}

	errorChan := make(chan error, 1)

	err = startRound(testProtoRound, testState, errorChan)
	if err == nil {
		t.Errorf("Expected error. Artificially created round " +
			"should make starting precomputing impossible")
	}

	if testState.GetRoundMap().GetRound(0).GetRoundState() == states.PRECOMPUTING {
		t.Errorf("Should not be in precomputing after artificially incrementign round")
	}
}

// Error path
func TestStartRound_BadNode(t *testing.T) {
	// Build params for scheduling
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build node list
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "")
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState

	}

	// Build pool
	testPool := &waitingPoll{
		pool:     nodeList,
		position: int(testParams.TeamSize),
	}

	roundID := NewRoundID(0)
	badState := round.NewState_Testing(roundID.Get(), states.COMPLETED, t)

	testProtoRound, err := createRound(testParams, testPool, roundID.Get(), testState)
	if err != nil {
		t.Errorf("Happy path of createRound failed: %v", err)
	}
	// Manually set the round of a node
	testProtoRound.nodeStateList[0].SetRound(badState)

	errorChan := make(chan error, 1)

	err = startRound(testProtoRound, testState, errorChan)
	if err == nil {
		t.Log(err)
		t.Errorf("Expected error. Artificially created round " +
			"should make starting precomputing impossible")
	}

}
