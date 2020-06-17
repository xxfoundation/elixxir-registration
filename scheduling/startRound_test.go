////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"crypto/rand"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"testing"
)

// Happy path
func TestStartRound(t *testing.T) {
	// Build params for scheduling
	// Build scheduling params
	testParams := Params{
		TeamSize:            8,
		BatchSize:           32,
		RandomOrdering:      false,
		Threshold:           1,
		Secure:              false,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey, "", "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build node list
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	// Build pool
	testPool := NewWaitingPool()

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], "Americas", "", "")
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testPool.Add(nodeState)
	}

	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}

	testProtoRound, err := createSecureRound(testParams, testPool, roundID, testState)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}

	errorChan := make(chan error, 1)

	err = startRound(testProtoRound, testState, errorChan)
	if err != nil {
		t.Errorf("Received error from startRound(): %v", err)
	}

	if testState.GetRoundMap().GetRound(1).GetRoundState() != states.PRECOMPUTING {
		t.Errorf("In unexpected state after round creation: %v",
			testState.GetRoundMap().GetRound(0).GetRoundState())
	}
}

// Error path
func TestStartRound_BadState(t *testing.T) {
	// Build params for scheduling
	testParams := Params{
		TeamSize:            8,
		BatchSize:           32,
		RandomOrdering:      false,
		Threshold:           1,
		Secure:              false,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey, "", "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build pool
	testPool := NewWaitingPool()

	// Build node list
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], "Americas", "", "")
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testPool.Add(nodeState)
	}

	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}

	// Manually set the state of the round
	badState := round.NewState_Testing(roundID, states.COMPLETED, t)
	testState.GetRoundMap().AddRound_Testing(badState, t)

	testProtoRound, err := createSecureRound(testParams, testPool, roundID, testState)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}

	errorChan := make(chan error, 1)

	err = startRound(testProtoRound, testState, errorChan)
	if err == nil {
		t.Errorf("Expected error. Artificially created round " +
			"should make starting precomputing impossible")
	}

	if testState.GetRoundMap().GetRound(1).GetRoundState() == states.PRECOMPUTING {
		t.Errorf("Should not be in precomputing after artificially incrementign round")
	}
}

// Error path
func TestStartRound_BadNode(t *testing.T) {
	// Build params for scheduling
	// Build params for scheduling
	testParams := Params{
		TeamSize:            8,
		BatchSize:           32,
		RandomOrdering:      false,
		Threshold:           1,
		Secure:              false,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey, "", "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build pool
	testPool := NewWaitingPool()

	// Build node list
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], "Americas", "", "")
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testPool.Add(nodeState)
	}

	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}
	badState := round.NewState_Testing(roundID, states.COMPLETED, t)

	testProtoRound, err := createSecureRound(testParams, testPool, roundID, testState)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}
	// Manually set the round of a node
	testProtoRound.NodeStateList[0].SetRound(badState)

	errorChan := make(chan error, 1)

	err = startRound(testProtoRound, testState, errorChan)
	if err == nil {
		t.Log(err)
		t.Errorf("Expected error. Artificially created round " +
			"should make starting precomputing impossible")
	}

}
