////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"crypto/rand"
	"github.com/golang-collections/collections/set"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"reflect"
	"testing"
	"time"
)

func TestNewWaitingPool(t *testing.T) {

	expectedPool := &waitingPool{
		pool:    set.New(),
		offline: set.New(),
	}

	// Create a pool
	receivedPool := NewWaitingPool()

	if !reflect.DeepEqual(expectedPool, receivedPool) {
		t.Errorf("New pool is not expected."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", expectedPool, receivedPool)
	}

	// Test that pool length has increased
	if receivedPool.pool.Len() != 0 {
		t.Errorf("Failed to insert node into pool. Was something modified?")
	}

}

func TestWaitingPool_Add(t *testing.T) {
	testPool := NewWaitingPool()

	// Make a node state
	expectedNode := setupNode(t, setupNodeMap(t), 0)

	// Place node into pool
	testPool.Add(expectedNode)

	// Test that the expected node is indeed inserted
	if !testPool.pool.Has(expectedNode) {
		t.Errorf("Pool doesn't have expected eleement")
	}

	// Test that pool length has increased
	if testPool.Len() != 1 || testPool.Len() != testPool.pool.Len() {
		t.Errorf("Failed to insert node into pool. Was something modified?")
	}
}

func TestWaitingPool_SetNodeToOnline(t *testing.T) {
	testPool := NewWaitingPool()

	// Make a node state
	oldNode := setupNode(t, setupNodeMap(t), 0)

	// Create a time that was long ago
	longAgo := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	// Set last poll to a really old time
	oldNode.SetLastPoll(longAgo, t)

	// Place ancient node into pool
	testPool.Add(oldNode)

	// Place node in online pool
	testPool.SetNodeToOnline(oldNode)

	// Make sure the offlien pool is now empty
	if testPool.OfflineLen() != 0 {
		t.Errorf("Offline pool expected to be empty. Actual size: %d", testPool.OfflineLen())
	}

	// make sure that the online pool is now non-empty
	if testPool.Len() != 1 {
		t.Errorf("Online pool expected to have one node inside. Actual size: %d", testPool.Len())
	}

}

func TestWaitingPool_PickNRandAtThreshold(t *testing.T) {
	testPool := NewWaitingPool()
	testState := setupNodeMap(t)

	totalNodes := 10
	requestedNodes := totalNodes / 2
	threshold := totalNodes / 2

	for i := 0; i < totalNodes; i++ {
		// Make a node state
		newNode := setupNode(t, testState, uint64(i))

		// Set last poll to a recent time
		newNode.SetLastPoll(time.Now(), t)

		// Place ancient node into pool
		testPool.Add(newNode)

	}

	nodeList, err := testPool.PickNRandAtThreshold(threshold, requestedNodes)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(nodeList) != 5 {
		t.Errorf("Node list not of expected length."+
			"\n\tExpected: %d: "+
			"\n\tReceived: %d", 5, len(nodeList))
	}

}

// Error path: does not meet threshold
func TestWaitingPool_PickNRandAtThreshold_ThresholdErr(t *testing.T) {
	testPool := NewWaitingPool()
	testState := setupNodeMap(t)

	totalNodes := 10
	threshold := totalNodes * 2
	requestedNodes := totalNodes / 2

	for i := 0; i < totalNodes; i++ {
		// Make a node state
		newNode := setupNode(t, testState, uint64(i))

		// Set last poll to a recent time
		newNode.SetLastPoll(time.Now(), t)

		// Place ancient node into pool
		testPool.Add(newNode)

	}

	_, err := testPool.PickNRandAtThreshold(threshold, requestedNodes)
	if err != nil {
		return
	}

	t.Errorf("Not meeting threshold should return an error: %v", err)

}

// Error path: Request more nodes than exist
func TestWaitingPool_PickNRandAtThreshold_NotEnoughNodesErr(t *testing.T) {
	testPool := NewWaitingPool()
	testState := setupNodeMap(t)

	totalNodes := 10
	threshold := totalNodes / 2
	requestedNodes := totalNodes * 2

	for i := 0; i < totalNodes; i++ {
		newNode := setupNode(t, testState, uint64(i))

		// Set last poll to a recent time
		newNode.SetLastPoll(time.Now(), t)

		// Place ancient node into pool
		testPool.Add(newNode)

	}

	_, err := testPool.PickNRandAtThreshold(threshold, requestedNodes)
	if err != nil {
		return
	}

	t.Errorf("Expected error case: "+
		"Should not be able to pick %d nodes when only %d exist", requestedNodes, totalNodes)

}

// Sets up a node state object
func setupNode(t *testing.T, testState *storage.NetworkState, newId uint64) *node.State {

	// Construct a node state
	nid := id.NewIdFromUInt(newId, id.Node, t)
	err := testState.GetNodeMap().AddNode(nid, "0", "", "", 0)
	if err != nil {
		t.Errorf("Failed to add node to state: %v", err)
	}
	// Retrieve an expected node
	return testState.GetNodeMap().GetNode(nid)
}

func setupNodeMap(t *testing.T) *storage.NetworkState {
	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	return testState
}
