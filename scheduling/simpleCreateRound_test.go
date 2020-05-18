///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

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
		TeamSize:            5,
		BatchSize:           32,
		RandomOrdering:      false,
		Threshold:           0,
		NodeCleanUpInterval: 3,
		Secure:              false,
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
	// Build pool
	// fixme: this test required a crafted (full) pool, which is no longer possible..

	testPool := NewWaitingPool()

	for i := 0; i < int(testParams.TeamSize); i++ {
		nid := id.NewIdFromUInt(uint64(i), id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testPool.Add(nodeState)
	}

	expectedTopology := connect.NewCircuit(nodeList)

	roundID := NewRoundID(0)

	testProtoRound, err := createSimpleRound(testParams, testPool, roundID.Get(), testState)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}

	if testProtoRound.ID != roundID.Get() {
		t.Errorf("ProtoRound's id returned unexpected value!"+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", roundID.Get(), testProtoRound.ID)
	}

	if !reflect.DeepEqual(testProtoRound.Topology, expectedTopology) {
		t.Errorf("ProtoRound's topology returned unexpected value!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedTopology, testProtoRound.Topology)
	}

	if testParams.BatchSize != testProtoRound.BatchSize {
		t.Errorf("ProtoRound's batchsize returned unexpected value!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", testParams.BatchSize, testProtoRound.BatchSize)

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
	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		// Input an invalid ordering to node
		err := testState.GetNodeMap().AddNode(nodeList[i], "BadNumber")
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	// Build pool
	// fixme: this test required a crafted (full) pool, which is no longer possible..

	testPool := NewWaitingPool()

	roundID := NewRoundID(0)

	// Invalid ordering will cause this to fail
	_, err = createSecureRound(testParams, testPool, roundID.Get(), testState)
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
		TeamSize:            10,
		BatchSize:           32,
		RandomOrdering:      true,
		Threshold:           1,
		Secure:              false,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	testPool := NewWaitingPool()

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testPool.Add(nodeState)
	}

	initialTopology := connect.NewCircuit(nodeList)

	roundID := NewRoundID(0)

	testProtoRound, err := createSecureRound(testParams, testPool, roundID.Get(), testState)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}

	// Check that shuffling has actually occurred
	// This has a chance to fail even when successful, however that chance is 1 in ~3.6 million
	if reflect.DeepEqual(initialTopology, testProtoRound.Topology) {
		t.Errorf("Highly unlikely initial topology identical to resulting after shuffling. " +
			"Possile shuffling is broken")
	}

	if testProtoRound.ID != roundID.Get() {
		t.Errorf("ProtoRound's id returned unexpected value!"+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", roundID.Get(), testProtoRound.ID)
	}

	if testParams.BatchSize != testProtoRound.BatchSize {
		t.Errorf("ProtoRound's batchsize returned unexpected value!"+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", testParams.BatchSize, testProtoRound.BatchSize)

	}

}
