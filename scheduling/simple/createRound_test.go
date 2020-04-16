///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"crypto/rand"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"reflect"
	"strconv"
	"testing"
)

// Happy path
func TestCreateRound_NonRandom(t *testing.T) {
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
	nodeList := make([]*id.Node, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewNodeFromUInt(i, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState

	}

	expectedTopology := connect.NewCircuit(nodeList)

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

	// todo: move this to start round testing
	//if testState.GetRoundMap().GetRound(0).GetRoundState() != states.PRECOMPUTING {
	//	t.Errorf("In unexpected state after round creation: %v",
	//		testState.GetRoundMap().GetRound(0).GetRoundState())
	//}


	if testProtoRound.ID != roundID.Get() {
		t.Errorf("ProtoRound's id returned unexpected value!" +
			"\n\tExpected: %d" +
			"\n\tReceived: %d", roundID.Get(), testProtoRound.ID)
	}

	if !reflect.DeepEqual(testProtoRound.topology, expectedTopology) {
		t.Errorf("ProtoRound's topology returned unexpected value!" +
			"\n\tExpected: %v" +
			"\n\tReceived: %v", expectedTopology, testProtoRound.topology)
	}

	if testParams.BatchSize != testProtoRound.batchSize {
		t.Errorf("ProtoRound's batchsize returned unexpected value!" +
			"\n\tExpected: %v" +
			"\n\tReceived: %v", testParams.BatchSize, testProtoRound.batchSize)

	}
	if !reflect.DeepEqual(testProtoRound.nodeStateList, nodeStateList) {
		t.Errorf("ProtoRound's nodeStateList returned unexpected value!" +
			"\n\tExpected: %v" +
			"\n\tReceived: %v", nodeStateList, testProtoRound.nodeStateList)

	}

}

// Error path: Provide a node ordering that is invalid
func TestCreateRound_BadOrdering(t *testing.T) {
	// Build scheduling params
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

	// Build a node list that will be invalid
	nodeList := make([]*id.Node, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewNodeFromUInt(i, t)
		// Input an invalid ordering to node
		err := testState.GetNodeMap().AddNode(nodeList[i], "BadNumber")
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	// Build pool
	testPool := &waitingPoll{
		pool:     nodeList,
		position: int(testParams.TeamSize),
	}

	roundID := NewRoundID(0)

	// Invalid ordering will cause this to fail
	_, err = createRound(testParams, testPool, roundID.Get(), testState)
	if err != nil {
		return
	}


	t.Errorf("Expected error case: passed in an ordering to nodes which were not numbers should result " +
		"in an error")

}

// Happy path for random ordering
func TestCreateRound_RandomOrdering(t *testing.T) {
	// Build scheduling params
	testParams := Params{
		TeamSize:       10,
		BatchSize:      32,
		RandomOrdering: true,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.Node, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewNodeFromUInt(i, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState

	}

	initialTopology := connect.NewCircuit(nodeList)

	testPool := &waitingPoll{
		pool:     nodeList,
		position: int(testParams.TeamSize),
	}

	roundID := NewRoundID(0)

	testProtoRound, err := createRound(testParams, testPool, roundID.Get(), testState)
	if err != nil {
		t.Errorf("Happy path of createRound failed: %v", err)
	}


	// Check that shuffling has actually occurred
	// This has a chance to fail even when successful, however that chance is 1 in ~3.6 million
	if reflect.DeepEqual(initialTopology, testProtoRound.topology) {
		t.Errorf("Highly unlikely initial topology identical to resulting after shuffling. " +
			"Possile shuffling is broken")
	}

	if testProtoRound.ID != roundID.Get() {
		t.Errorf("ProtoRound's id returned unexpected value!" +
			"\n\tExpected: %d" +
			"\n\tReceived: %d", roundID.Get(), testProtoRound.ID)
	}

	if testParams.BatchSize != testProtoRound.batchSize {
		t.Errorf("ProtoRound's batchsize returned unexpected value!" +
			"\n\tExpected: %v" +
			"\n\tReceived: %v", testParams.BatchSize, testProtoRound.batchSize)

	}
	if !reflect.DeepEqual(testProtoRound.nodeStateList, nodeStateList) {
		t.Errorf("ProtoRound's nodeStateList returned unexpected value!" +
			"\n\tExpected: %v" +
			"\n\tReceived: %v", nodeStateList, testProtoRound.nodeStateList)

	}

}
