////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"crypto/rand"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"testing"
	"time"
)

// Happy path for transitioning to waiting
func TestHandleNodeStateChance_Waiting(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf(err.Error())
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	// Set a round for the node in order to fully test the code path for
	//  a waiting transition
	roundState := round.NewState_Testing(roundID, 0, nil, t)
	_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.NOT_STARTED,
		ToActivity:   current.WAITING}

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	roundTracker := NewRoundTracker()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, roundTracker)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}

}

// Tests that WAITING sets the node to online when the node is in the offline
// pool.
func TestHandleNodeStateChance_Waiting_SetNodeToOnline(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	// Set a round for the node in order to fully test the code path for
	//  a waiting transition
	roundState := round.NewState_Testing(roundID, 0, nil, t)
	_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.NOT_STARTED,
		ToActivity:   current.WAITING,
		FromStatus:   node.Inactive,
		ToStatus:     node.Active,
	}

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	roundTracker := NewRoundTracker()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, roundTracker)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}

	if !testPool.pool.Has(testState.GetNodeMap().GetNode(nodeList[0])) ||
		testPool.offline.Has(testState.GetNodeMap().GetNode(nodeList[0])) {
		t.Errorf("The node was not set to online.")
	}

}

// Happy path
func TestHandleNodeStateChance_Standby(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize-1)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}
	circuit := connect.NewCircuit(nodeList)

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	roundState, err := testState.GetRoundMap().AddRound(roundID, testParams.BatchSize, 8, 5*time.Minute, circuit)
	if err != nil {
		t.Errorf("Failed to add round: %v", err)
	}

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	for i := range nodeList {
		testUpdate := node.UpdateNotification{
			Node:         nodeList[i],
			FromActivity: current.NOT_STARTED,
			ToActivity:   current.WAITING,
		}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()
		testTracker := NewRoundTracker()
		err = HandleNodeUpdates(testUpdate, testPool, testState, 0, testTracker)
		if err != nil {
			t.Errorf("Waiting pool is full for %d: %v", i, err)
		}
	}

	// Iterate through all the nodes so that all the nodes are ready for transition
	for i := range nodeList {
		_ = testState.GetNodeMap().GetNode(nodeList[i]).SetRound(roundState)

		testUpdate := node.UpdateNotification{
			Node:         nodeList[i],
			FromActivity: current.WAITING,
			ToActivity:   current.STANDBY}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err = HandleNodeUpdates(testUpdate, testPool, testState, 0, nil)
		if err != nil {
			t.Errorf("Error in standby happy path: %v", err)
		}

	}

}

// Error path: Do not give a round to the nodes
func TestHandleNodeStateChance_Standby_NoRound(t *testing.T) {

	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	//roundID := NewRoundID(0)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	// Explicitly don't give node a round to reach an error state
	//roundState := round.NewState_Testing(roundID.Get(), 0, t)
	//_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Iterate through all the nodes so that all the nodes are ready for transition
	for i := range nodeList {
		testUpdate := node.UpdateNotification{
			Node:         nodeList[i],
			FromActivity: current.WAITING,
			ToActivity:   current.STANDBY}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()
		testTracker := NewRoundTracker()
		err := HandleNodeUpdates(testUpdate, testPool, testState, 0, testTracker)
		if err == nil {
			t.Errorf("Expected error for %d was not received. Node should not have round", i)
		}

	}

}

// Happy path
func TestHandleNodeUpdates_Completed(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("test", "password",
		"regCodes", "", "")
	if err != nil {
		t.Errorf("%+v", err)
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}
	circuit := connect.NewCircuit(nodeList)

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	roundState, err := testState.GetRoundMap().AddRound(roundID, testParams.BatchSize, 8, 5*time.Minute, circuit)
	if err != nil {
		t.Errorf("Failed to add round: %v", err)
	}

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()
	testTracker := NewRoundTracker()

	for i := range nodeList {
		testUpdate := node.UpdateNotification{
			Node:         nodeList[i],
			FromActivity: current.NOT_STARTED,
			ToActivity:   current.WAITING,
		}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err := HandleNodeUpdates(testUpdate, testPool, testState, 0, testTracker)
		if err != nil {
			t.Errorf("Waiting pool is full for %d: %v", i, err)
		}
	}

	// Iterate through all the nodes so that all the nodes are ready for transition
	for i := range nodeList {
		_ = testState.GetNodeMap().GetNode(nodeList[i]).SetRound(roundState)

		testUpdate := node.UpdateNotification{
			Node:         nodeList[i],
			FromActivity: current.REALTIME,
			ToActivity:   current.COMPLETED}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err := HandleNodeUpdates(testUpdate, testPool, testState, 0, testTracker)
		if err != nil {
			t.Errorf("Expected happy path for completed: %v", err)
		}
	}
}

// Error path: attempt to handle a node transition when nodes never had rounds
func TestHandleNodeUpdates_Completed_NoRound(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	//roundID := NewRoundID(0)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	// Iterate through all the nodes so that all the nodes are ready for transition
	for i := range nodeList {
		testUpdate := node.UpdateNotification{
			Node:         nodeList[i],
			FromActivity: current.WAITING,
			ToActivity:   current.COMPLETED}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()
		testTracker := NewRoundTracker()

		err := HandleNodeUpdates(testUpdate, testPool, testState, 0, testTracker)
		if err == nil {
			t.Errorf("Expected error for %d was not received. Node should not have round", i)
		}
	}
}

func TestHandleNodeUpdates_Error(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf(err.Error())
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}
	topology := connect.NewCircuit(nodeList)

	// Set a round for the node in order to fully test the code path for
	//  a waiting transition
	roundState := round.NewState_Testing(roundID, 0, topology, t)
	_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.WAITING,
		ToActivity:   current.ERROR,
		Error: &mixmessages.RoundError{
			Id:     0,
			NodeId: id.NewIdFromString("test", id.Node, t).Bytes(),
			Error:  "test",
		},
	}
	testState.GetNodeMap().GetNode(testUpdate.Node).GetPollingLock().Lock()

	testTracker := NewRoundTracker()

	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, testTracker)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}
}

// Happy path: Test that a node with a banned update status are removed from the pool
func TestHandleNodeUpdates_BannedNode(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	testPool := NewWaitingPool()

	// Build mock nodes and place in map
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}

		// Add node to pool
		ns := testState.GetNodeMap().GetNode(nodeList[i])
		nodeStateList = append(nodeStateList, ns)
		testPool.Add(ns)
	}

	// Test that a node with no round gets removed from the pool
	testUpdate := node.UpdateNotification{
		Node:     nodeList[0],
		ToStatus: node.Banned,
	}
	testTracker := NewRoundTracker()

	// Ban the first node in the state map
	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, testTracker)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}

	if testPool.Len() == int(testParams.TeamSize) {
		t.Errorf("Node expected to be banned, decreasing pool size."+
			"\n\tExpected size: %v"+
			"\n\tReceived size: %v", testParams.TeamSize-1, testPool.Len())
	}

	topology := connect.NewCircuit(nodeList)

	r := round.NewState_Testing(42, 0, topology, t)

	// Get a node and set the round of the node
	ns := testState.GetNodeMap().GetNode(nodeList[1])
	err = ns.SetRound(r)
	if err != nil {
		t.Errorf("Unable to set round for mock node: %v", err)
	}

	// Craft a node update
	testUpdate = node.UpdateNotification{
		Node:     nodeList[1],
		ToStatus: node.Banned,
	}

	// Ban the the second node in the state map
	testState.GetNodeMap().GetNode(nodeList[1]).GetPollingLock().Lock()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, testTracker)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}

	// Test that a node with a round gets has it's round unset
	ok, receivedRound := ns.GetCurrentRound()
	if ok {
		t.Errorf("Did not expect node with round after being banned."+
			"\n\tExpected nil round."+
			"\n\tReceived: %v", receivedRound)
	}

}

// Happy path
func TestKillRound(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	testPool := NewWaitingPool()

	// Build mock nodes and place in map
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}

		// Add node to pool
		ns := testState.GetNodeMap().GetNode(nodeList[i])
		nodeStateList = append(nodeStateList, ns)
		testPool.Add(ns)
	}

	topology := connect.NewCircuit(nodeList)

	r := round.NewState_Testing(42, 0, topology, t)

	re := &mixmessages.RoundError{
		Id:     0,
		NodeId: nil,
		Error:  "test",
	}

	tesTracker := NewRoundTracker()

	err = killRound(testState, r, re, tesTracker, nil)
	if err != nil {
		t.Errorf("Unexpected error in happy path: %v", err)
	}
}

// Tests that the Precomputing case of HandleNodeUpdates produces the correct
// error when there is no round.
func TestHandleNodeUpdates_Precomputing_RoundError(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.WAITING,
		ToActivity:   current.PRECOMPUTING,
	}

	expectedErr := "Node " + nodeList[0].String() + " without round should " +
		"not be moving to the PRECOMPUTING state"

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	roundTracker := NewRoundTracker()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, roundTracker)
	if err == nil {
		t.Errorf("HandleNodeUpdates() did not produce the expected error when"+
			"there is no round.\n\texpected: %v\n\treceived: %v",
			expectedErr, err)
	}
}

// Tests happy path of the Realtime case of HandleNodeUpdates.
func TestHandleNodeUpdates_Realtime(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	// Set a round for the node in order to fully test the code path for
	//  a waiting transition
	roundState := round.NewState_Testing(roundID, 0, nil, t)
	_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.STANDBY,
		ToActivity:   current.REALTIME}

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	roundTracker := NewRoundTracker()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, roundTracker)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}
}

// Tests that the Realtime case of HandleNodeUpdates produces the correct
// error when there is no round.
func TestHandleNodeUpdates_Realtime_RoundError(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.STANDBY,
		ToActivity:   current.REALTIME,
	}

	expectedErr := "Node " + nodeList[0].String() + " without round should " +
		"not be moving to the REALTIME state"

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	roundTracker := NewRoundTracker()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, roundTracker)
	if err == nil {
		t.Errorf("HandleNodeUpdates() did not produce the expected error when"+
			"there is no round.\n\texpected: %v\n\treceived: %v",
			expectedErr, err)
	}
}

// Tests that the Realtime case of HandleNodeUpdates produces an error when the
// round can't update.
func TestHandleNodeUpdates_Realtime_UpdateError(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	// Set a round for the node in order to fully test the code path for
	//  a waiting transition
	roundState := round.NewState_Testing(roundID, 5, nil, t)
	_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.STANDBY,
		ToActivity:   current.REALTIME}

	expectedErr := "Node " + nodeList[0].String() + " without round should " +
		"not be moving to the REALTIME state"

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	roundTracker := NewRoundTracker()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, roundTracker)
	if err == nil {
		t.Errorf("HandleNodeUpdates() did not produce the expected error."+
			"\n\texpected: %v\n\treceived: %v",
			expectedErr, err)
	}
}

// Tests that HandleNodeUpdates() returns nil when there is a round error.
func TestHandleNodeUpdates_RoundErrored(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)


	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	// Set a round for the node in order to fully test the code path for
	//  a waiting transition
	roundState := round.NewState_Testing(roundID, 6, nil, t)
	_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.STANDBY,
		ToActivity:   current.REALTIME}

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	roundTracker := NewRoundTracker()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0, roundTracker)
	if err != nil {
		t.Errorf("HandleNodeUpdates() recieved an unexpected error"+
			"\n\texpected: %v\n\treceived: %v",
			nil, err)
	}
}

// Tests happy path of the NOT_STARTED case of HandleNodeUpdates.
func TestHandleNodeUpdates_NOT_STARTED(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.NOT_STARTED,
		ToActivity:   current.NOT_STARTED}

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	err = HandleNodeUpdates(testUpdate, nil, testState, 0, nil)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}
}
