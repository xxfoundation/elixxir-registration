////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"crypto/rand"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"strconv"
	"testing"
)

// Happy path for transitioning to waiting
func TestHandleNodeStateChance_Waiting(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	roundID := NewRoundID(0).Get()

	// Set a round for the node in order to fully test the code path for
	//  a waiting transition
	roundState := round.NewState_Testing(roundID, 0, t)
	_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.NOT_STARTED,
		ToActivity:   current.WAITING}

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()

	err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
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
	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize-1)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}
	circuit := connect.NewCircuit(nodeList)

	roundID := NewRoundID(0)

	roundState, err := testState.GetRoundMap().AddRound(roundID.Get(), testParams.BatchSize, circuit)
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

		err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
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

		err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
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

	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
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

		err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
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

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}
	circuit := connect.NewCircuit(nodeList)

	roundID := NewRoundID(0)

	roundState, err := testState.GetRoundMap().AddRound(roundID.Get(), testParams.BatchSize, circuit)
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

		err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
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

		err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
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

	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
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

		err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
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

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.ID, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewIdFromUInt(i, id.Node, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	roundID := NewRoundID(0).Get()

	// Set a round for the node in order to fully test the code path for
	//  a waiting transition
	roundState := round.NewState_Testing(roundID, 0, t)
	_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	testUpdate := node.UpdateNotification{
		Node:         nodeList[0],
		FromActivity: current.WAITING,
		ToActivity:   current.ERROR}
	testState.GetNodeMap().GetNode(testUpdate.Node).GetPollingLock().Lock()

	err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
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

	testState, err := storage.NewState(privKey)
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
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
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

	// Ban the first node in the state map
	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}

	if testPool.Len() == int(testParams.TeamSize) {
		t.Errorf("Node expected to be banned, decreasing pool size."+
			"\n\tExpected size: %v"+
			"\n\tReceived size: %v", testParams.TeamSize-1, testPool.Len())
	}

	r := round.NewState_Testing(42, 0, t)

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
	err = HandleNodeUpdates(testUpdate, testPool, testState, 0)
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

	testState, err := storage.NewState(privKey)
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
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}

		// Add node to pool
		ns := testState.GetNodeMap().GetNode(nodeList[i])
		nodeStateList = append(nodeStateList, ns)
		testPool.Add(ns)
	}

	r := round.NewState_Testing(42, 0, t)

	ns := testState.GetNodeMap().GetNode(nodeList[0])

	//
	err = killRound(testState, r, ns)
	if err != nil {
		t.Errorf("Unexpected error in happy path: %v", err)
	}
}
