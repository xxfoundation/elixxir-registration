////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"gitlab.com/elixxir/primitives/id"
	"reflect"
	"testing"
)

// Happy path
func TestNewPool(t *testing.T) {
	testingSize := 5
	receivedPool := newWaitingPool(testingSize)

	if len(receivedPool.pool) != testingSize {
		t.Errorf("Pool is not of expected size."+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", testingSize, len(receivedPool.pool))
	}

	if receivedPool.position != 0 {
		t.Errorf("Newly created pool should have a position of %d. "+
			"Received position is: %v", 0, receivedPool.position)
	}
}

// Happy path
func TestWaitingPoll_Add(t *testing.T) {
	testingSize := 5
	testPoll := newWaitingPool(testingSize)

	// node id with arbitrary initializer
	for i := 0; i < testingSize; i++ {
		nid := id.NewNodeFromUInt(uint64(i), t)

		err := testPoll.Add(nid)
		if err != nil {
			t.Errorf("Should not receive error when adding first node to pool: %v", err)
		}
		if testPoll.position-1 != i {
			t.Errorf("Position should increment when adding one node to pool."+
				"\n\tPosition: %d"+
				"\n\tExpected: %d", testPoll.position, i)
		}
	}

}

// Error path: Should not be able to insert a node past maximum
//  pool size
func TestWaitingPoll_Add_Full(t *testing.T) {
	testingSize := 5
	testPoll := newWaitingPool(testingSize)

	for i := 0; i < testingSize; i++ {
		nid := id.NewNodeFromUInt(uint64(i), t)
		testPoll.Add(nid)
	}

	// Attempt to push an additional node in
	nid := id.NewNodeFromUInt(uint64(5), t)

	err := testPoll.Add(nid)
	if err == nil {
		t.Errorf("Should not be able to add to a full pool")
	}

	if testPoll.position != testingSize {
		t.Errorf("Position should not increment past the maximin size")
	}

}

// Happy path
func TestWaitingPoll_Clear(t *testing.T) {
	testingSize := 5
	testPoll := newWaitingPool(testingSize)

	nodeList := make([]*id.Node, testingSize)
	for i := 0; i < testingSize; i++ {
		nid := id.NewNodeFromUInt(uint64(i), t)
		testPoll.Add(nid)
		nodeList[i] = nid
	}

	receivedList := testPoll.Clear()

	if !reflect.DeepEqual(nodeList, receivedList) {
		t.Errorf("Node list received from clear did not match expected node list."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", nodeList, receivedList)
	}

	emptyNodeList := make([]*id.Node, testingSize)
	if !reflect.DeepEqual(testPoll.pool, emptyNodeList) {
		t.Errorf("After clearing, waiting pool should not contain any nodes."+
			"\n\tExpected: %v"+
			"\n\tReceived: %v", emptyNodeList, testPoll.pool)
	}

	if testPoll.position != 0 {
		t.Errorf("Waiting pool's position should be reset after clear."+
			"Expected: %v"+
			"Position: %v", 0, testPoll.position)
	}
}

// Happy path
func TestWaitingPoll_Size(t *testing.T) {
	testingSize := 5
	testPoll := newWaitingPool(testingSize)

	nid := id.NewNodeFromUInt(uint64(1), t)
	testPoll.Add(nid)

	receivedSize := testPoll.Size()

	if receivedSize != testingSize {
		t.Errorf("Pool not of expected size")
	}
}

// Happy path
func TestWaitingPoll_Len(t *testing.T) {
	testingSize := 5
	testPoll := newWaitingPool(testingSize)

	// node id with arbitrary initializer
	for i := 0; i < testingSize; i++ {
		nid := id.NewNodeFromUInt(uint64(i), t)

		err := testPoll.Add(nid)
		if err != nil {
			t.Errorf("Should not receive error when adding first node to pool: %v", err)
		}

		receivedLen := testPoll.Len()
		if receivedLen != i+1 {
			t.Errorf("Expected position not received. "+
				"\n\tExpected: %d"+
				"\n\tReceived: %d", i+1, receivedLen)
		}
	}
}
