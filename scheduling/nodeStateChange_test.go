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
		Node: nodeList[0],
		FromActivity: current.NOT_STARTED,
		ToActivity:   current.WAITING}

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()

	err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}
}

// Happy path for transitioning to waiting
func TestHandleNodeStateChance_WaitingError(t *testing.T) {
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

	// Unfilled poll s.t. we can add a node to the waiting pool
	// fixme: this test required a crafted (full) pool, which is no longer possible..

	testPool := NewWaitingPool()
	testUpdate := node.UpdateNotification{
		Node: nodeList[0],
		FromActivity: current.NOT_STARTED,
		ToActivity:   current.WAITING}

	testState.GetNodeMap().GetNode(nodeList[0]).GetPollingLock().Lock()

	err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
	if err != nil {
		return
	}

	t.Errorf("Should fail when trying to insert node into a full pool")
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
			Node: nodeList[i],
			FromActivity: current.NOT_STARTED,
			ToActivity:   current.WAITING,
		}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
		if err != nil {
			t.Errorf("Waiting pool is full for %d: %v", i, err)
		}
	}

	// Iterate through all the nodes so that all the nodes are ready for transition
	for i := range nodeList {
		_ = testState.GetNodeMap().GetNode(nodeList[i]).SetRound(roundState)

		testUpdate := node.UpdateNotification{
			Node: nodeList[i],
			FromActivity: current.WAITING,
			ToActivity:   current.STANDBY}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
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
	// fixme: this test required a crafted (full) pool, which is no longer possible..

	testPool := NewWaitingPool()

	// Explicitly don't give node a round to reach an error state
	//roundState := round.NewState_Testing(roundID.Get(), 0, t)
	//_ = testState.GetNodeMap().GetNode(nodeList[0]).SetRound(roundState)

	// Iterate through all the nodes so that all the nodes are ready for transition
	for i := range nodeList {
		testUpdate := node.UpdateNotification{
			Node: nodeList[i],
			FromActivity: current.WAITING,
			ToActivity:   current.STANDBY}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
		if err == nil {
			t.Errorf("Expected error for %d was not received. Node should not have round", i)
		}

	}

}

// Happy path
func TestHandleNodeStateChange_Completed(t *testing.T) {
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
			Node: nodeList[i],
			FromActivity: current.NOT_STARTED,
			ToActivity:   current.WAITING,
		}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
		if err != nil {
			t.Errorf("Waiting pool is full for %d: %v", i, err)
		}
	}

	// Iterate through all the nodes so that all the nodes are ready for transition
	for i := range nodeList {
		_ = testState.GetNodeMap().GetNode(nodeList[i]).SetRound(roundState)

		testUpdate := node.UpdateNotification{
			Node: nodeList[i],
			FromActivity: current.REALTIME,
			ToActivity:   current.COMPLETED}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
		if err != nil {
			t.Errorf("Expected happy path for completed: %v", err)
		}

	}
}

// Error path: attempt to handle a node transition when nodes never had rounds
func TestHandleNodeStateChange_Completed_NoRound(t *testing.T) {
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

	// fixme: this test required a crafted (full) pool, which is no longer possible..
	// Unfilled poll s.t. we can add a node to the waiting pool
	testPool := NewWaitingPool()

	// Iterate through all the nodes so that all the nodes are ready for transition
	for i := range nodeList {
		testUpdate := node.UpdateNotification{
			Node: nodeList[i],
			FromActivity: current.WAITING,
			ToActivity:   current.COMPLETED}

		testState.GetNodeMap().GetNode(nodeList[i]).GetPollingLock().Lock()

		err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
		if err == nil {
			t.Errorf("Expected error for %d was not received. Node should not have round", i)
		}

	}
}

func TestHandleNodeStateChange_Error(t *testing.T) {
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
		Node: nodeList[0],
		FromActivity: current.WAITING,
		ToActivity:   current.ERROR}
	testState.GetNodeMap().GetNode(testUpdate.Node).GetPollingLock().Lock()

	err = HandleNodeStateChange(testUpdate, testPool, testState, 0)
	if err != nil {
		t.Errorf("Happy path received error: %v", err)
	}
}
