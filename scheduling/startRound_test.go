////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package scheduling

import (
	"crypto/rand"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/region"
	mathRand "math/rand"
	"testing"
)

// Happy path
func TestStartRound(t *testing.T) {
	// Build params for scheduling
	// Build scheduling params
	testParams := Params{
		TeamSize:            8,
		BatchSize:           32,
		Threshold:           0.3,
		NodeCleanUpInterval: 3,
	}

	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf(err.Error())
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", "", region.GetCountryBins())
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
		err := testState.GetNodeMap().AddNode(nodeList[i], "US", "", "", 0)
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
	prng := mathRand.New(mathRand.NewSource(42))

	testProtoRound, err := createSecureRound(testParams, testPool, int(testParams.Threshold*float64(testParams.TeamSize)), roundID, testState, prng)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}

	testTracker := NewRoundTracker()

	_, err = startRound(testProtoRound, testState, testTracker)
	if err != nil {
		t.Errorf("Received error from startRound(): %v", err)
	}

	r, exists := testState.GetRoundMap().GetRound(1)
	if !exists {
		t.Errorf("Created round does not exist when it should")
	}

	if r.GetRoundState() != states.PRECOMPUTING {
		t.Errorf("In unexpected state after round creation: %v",
			r.GetRoundState())
	}
}

// Error path
func TestStartRound_BadState(t *testing.T) {
	// Build params for scheduling
	testParams := Params{
		TeamSize:            8,
		BatchSize:           32,
		Threshold:           0.3,
		NodeCleanUpInterval: 3,
	}

	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf(err.Error())
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", "", region.GetCountryBins())
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
		err := testState.GetNodeMap().AddNode(nodeList[i], "US", "", "", 0)
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
	badState := round.NewState_Testing(roundID, states.COMPLETED, nil, t)
	testState.GetRoundMap().AddRound_Testing(badState, t)
	prng := mathRand.New(mathRand.NewSource(42))

	testProtoRound, err := createSecureRound(testParams, testPool, int(testParams.Threshold*float64(testParams.TeamSize)), roundID, testState, prng)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}

	testTracker := NewRoundTracker()

	_, err = startRound(testProtoRound, testState, testTracker)
	if err == nil {
		t.Errorf("Expected error. Artificially created round " +
			"should make starting precomputing impossible")
	}

	r, exists := testState.GetRoundMap().GetRound(1)
	if !exists {
		t.Errorf("round should exist")
	}

	if r.GetRoundState() == states.PRECOMPUTING {
		t.Errorf("Should not be in precomputing after artificially incrementign round")
	}
}

// Error path
func TestStartRound_BadNode(t *testing.T) {
	// Build params for scheduling
	testParams := Params{
		TeamSize:            8,
		BatchSize:           32,
		Threshold:           0.3,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", "", region.GetCountryBins())
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
		err := testState.GetNodeMap().AddNode(nodeList[i], "US", "", "", 0)
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
	badState := round.NewState_Testing(roundID, states.COMPLETED, nil, t)
	prng := mathRand.New(mathRand.NewSource(42))

	testProtoRound, err := createSecureRound(testParams, testPool, int(testParams.Threshold*float64(testParams.TeamSize)), roundID, testState, prng)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}
	// Manually set the round of a node
	testProtoRound.NodeStateList[0].SetRound(badState)
	testTracker := NewRoundTracker()

	_, err = startRound(testProtoRound, testState, testTracker)
	if err == nil {
		t.Log(err)
		t.Errorf("Expected error. Artificially created round " +
			"should make starting precomputing impossible")
	}

}
