///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"crypto/rand"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/region"
	mathRand "math/rand"
	"reflect"
	"testing"
)

// Happy path
func TestCreateRound_Random(t *testing.T) {
	// Build params for scheduling
	testParams := Params{
		TeamSize:            5,
		BatchSize:           32,
		Threshold:           0,
		NodeCleanUpInterval: 3,
		Secure:              false,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", region.GetCountryBins(), nil)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build node list
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)

	// Build pool
	testPool := NewWaitingPool()

	for i := 0; i < int(testParams.TeamSize); i++ {
		nid := id.NewIdFromUInt(uint64(i), id.Node, t)
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
	testProtoRound, err := createSecureRound(testParams, testPool, roundID, testState, prng)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}

	if testProtoRound.ID != roundID {
		t.Errorf("ProtoRound's id returned unexpected value!"+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", roundID, testProtoRound.ID)
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
		TeamSize:  5,
		BatchSize: 32,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", region.GetCountryBins(), nil)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build a node list that will be invalid
	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		// Input an invalid ordering to node
		err := testState.GetNodeMap().AddNode(nodeList[i], "BadNumber", "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	// Build pool
	testPool := NewWaitingPool()

	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}

	// Invalid ordering will cause this to fail
	prng := mathRand.New(mathRand.NewSource(42))
	_, err = createSimpleRound(testParams, testPool, roundID, testState, prng)
	if err != nil {
		return
	}

	t.Errorf("Expected error case: passed in an ordering to nodes which were not numbers should result " +
		"in an error")

}

// Test that the system semi-optimal gets done when both
// random ordering and semioptimal ordering are set to true
func TestCreateSimpleRound_SemiOptimal(t *testing.T) {
	// Build scheduling params
	testParams := Params{
		TeamSize:            9,
		BatchSize:           32,
		Threshold:           1,
		Secure:              false,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", region.GetCountryBins(), nil)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	testPool := NewWaitingPool()

	// Craft regions for nodes
	regions := []string{"CR", "GB", "SK",
		"HR", "IQ", "BF", "RU", "CX"}

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		// Randomize the regions of the nodes
		index := mathRand.Intn(8)

		// Generate a test id
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid

		// Add the node to that node map
		// Place the node in a random region
		err := testState.GetNodeMap().AddNode(nodeList[i], regions[index], "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}

		// Add the node to the pool
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testPool.Add(nodeState)
	}

	initialTopology := connect.NewCircuit(nodeList)

	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}
	prng := mathRand.New(mathRand.NewSource(42))

	testProtoRound, err := createSimpleRound(testParams, testPool, roundID, testState, prng)
	if err != nil {
		t.Errorf("Happy path of createSimpleRound failed: %v", err)
	}

	// Check that shuffling has actually occurred, should not be the initial topology
	// which is inefficient
	if reflect.DeepEqual(initialTopology, testProtoRound.Topology) {
		t.Errorf("Highly unlikely initial topology identical to resulting after shuffling. " +
			"Possile shuffling is broken")
	}

}

// Test that the system semi-optimal gets done when both
// random ordering and semioptimal ordering are set to true
func TestCreateSimpleRound_SemiOptimal_BadRegion(t *testing.T) {
	// Build scheduling params
	testParams := Params{
		TeamSize:            9,
		BatchSize:           32,
		Threshold:           1,
		Secure:              false,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", region.GetCountryBins(), nil)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	testPool := NewWaitingPool()

	badRegion := "Mars"

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		// Generate a test id
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid

		// Add the node to that node map
		// Place the node in a random region
		err := testState.GetNodeMap().AddNode(nodeList[i], badRegion, "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}

		// Add the node to the pool
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testPool.Add(nodeState)
	}

	// Generate round id
	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}
	prng := mathRand.New(mathRand.NewSource(42))

	_, err = createSimpleRound(testParams, testPool, roundID, testState, prng)
	if err != nil {
		return
	}

	t.Errorf("Expected error path: Test should fail when receiving bad region %v!", badRegion)

}
