package scheduling

import (
	"container/ring"
	"crypto/rand"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"strconv"
	"testing"
)

// Happy path
func TestCreateRound(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            10,
		BatchSize:           32,
		RandomOrdering:      true,
		Threshold:           1,
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

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i%5)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testpool.Add(nodeState)
	}

	roundID := NewRoundID(0)

	_, err = createSecureRound(testParams, testpool, roundID.Get(), testState)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
}

func TestCreateRound_Error_NotEnoughForTeam(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            10,
		BatchSize:           32,
		RandomOrdering:      true,
		Threshold:           5,
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

	// Do not make a teamsize amount of nodes
	for i := uint64(0); i < 5; i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i%5)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testpool.Add(nodeState)
	}

	roundID := NewRoundID(0)

	_, err = createSecureRound(testParams, testpool, roundID.Get(), testState)
	if err != nil {
		return
	}

	t.Errorf("Expected error path: Number of nodes in pool" +
		" shouldn't be enough for a team size")

}

func TestCreateRound_Error_NotEnoughForThreshold(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            10,
		BatchSize:           32,
		RandomOrdering:      true,
		Threshold:           25,
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

	// Do not make a teamsize amount of nodes
	for i := uint64(0); i < 5; i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i%5)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testpool.Add(nodeState)
	}

	roundID := NewRoundID(0)

	_, err = createSecureRound(testParams, testpool, roundID.Get(), testState)
	if err != nil {
		return
	}

	t.Errorf("Expected error path: Number of nodes in pool" +
		" shouldn't be enough for threshold")

}

// Test that a team of 4 nodes is assembled in an efficient order
func TestCreateRound_EfficientTeam(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            4,
		BatchSize:           32,
		RandomOrdering:      true,
		Threshold:           2,
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

	// Craft regions for nodes
	regions := []int{0, 1, 2, 3}

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(regions[i]))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testpool.Add(nodeState)
	}

	roundID := NewRoundID(0)

	testProtoRound, err := createSecureRound(testParams, testpool, roundID.Get(), testState)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	ourRing := ring.New(int(testParams.TeamSize))
	var regionOrder []int
	// Ideal: 0 -> 1 -> 3 -> 2 (with any starting node)
	for _, n := range testProtoRound.NodeStateList {
		order, err := strconv.Atoi(n.GetOrdering())
		if err != nil {
			t.Errorf("Failed to convert node's order. Ordering: %s", n.GetOrdering())
		}
		ourRing.Value = order
		ourRing = ourRing.Next()
		regionOrder = append(regionOrder, order)
	}

	// Ideal iteration(s). It is possible that the ideal
	// order can go in 'reverse' order, as it is just a loop
	// We have to check the outputted order to see if it conforms
	// to either order
	idealOrder := []int{0, 2, 3, 1}
	idealOrderRev := []int{0, 1, 3, 2}

	var isReverse bool

	// Make the 0 value the head of the ring buffer
	for ourRing.Value != 0 {
		ourRing = ourRing.Next()
	}

	// Check if in the "reverse" order
	if ourRing.Next().Value == idealOrderRev[1] {
		isReverse = true
	}

	// Parse the buffer for correctness depending on order
	if isReverse {
		checkReverseOrder(idealOrderRev, regionOrder, ourRing, t)
	} else {
		checkOrder(idealOrder, regionOrder, ourRing, t)
	}

}

func checkReverseOrder(idealOrder, regionOrder []int, ourRing *ring.Ring, t *testing.T) {
	for j := 0; j < len(idealOrder); j++ {
		if ourRing.Value != idealOrder[j] {
			t.Errorf("Round made with innefficient order."+
				"\n\tExpected: %d"+
				"\n\tReceived: %d ", idealOrder[j], ourRing.Value)
			t.Logf("Actual order of nodes: %v", regionOrder)
			t.FailNow()
		}
		ourRing = ourRing.Next()
	}
}

func checkOrder(idealOrder, regionOrder []int, ourRing *ring.Ring, t *testing.T) {
	// Check that the order is expected (ie an efficient team)
	for j := 0; j < len(idealOrder); j++ {
		if ourRing.Value != idealOrder[j] {
			t.Errorf("Round made with innefficient order."+
				"\n\tExpected: %d"+
				"\n\tReceived: %d ", idealOrder[j], ourRing.Value)
			t.Logf("Actual order of nodes: %v", regionOrder)
			t.FailNow()
		}
		ourRing = ourRing.Next()
	}

}
