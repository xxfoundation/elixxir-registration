////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"strings"
	"testing"
	"time"
)

//tests that newStateMap is correct
func TestNewStateMap(t *testing.T) {
	sm := NewStateMap()

	if sm.nodeStates == nil {
		t.Errorf("Internal map not initilized")
	}
}

//Tests a Node is added correctly to the state map when it is
func TestStateMap_AddNode_Happy(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.ID]*State),
	}

	nid := id.NewIdFromUInt(2, id.Node, t)

	err := sm.AddNode(nid, "", "", "", 0)

	if err != nil {
		t.Errorf("Error returned on valid addition of Node: %s", err)
	}

	n := sm.nodeStates[*nid]

	if n.activity != current.NOT_STARTED {
		t.Errorf("New Node state has wrong activity; "+
			"Expected: %s, Recieved: %s", current.NOT_STARTED, n.activity)
	}

	if n.currentRound != nil {
		t.Errorf("New Node has a curent round set incorrectly")
	}
}

//Tests a Node is added correctly to the state map when it is
func TestStateMap_AddNode_Invalid(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.ID]*State),
	}

	nid := id.NewIdFromUInt(2, id.Node, t)
	r := round.NewState_Testing(42, 0, nil, t)

	sm.nodeStates[*nid] = &State{
		activity:     current.WAITING,
		currentRound: r,
		lastPoll:     time.Now(),
	}

	time.Sleep(1 * time.Millisecond)

	err := sm.AddNode(nid, "", "", "", 0)

	if err == nil {
		t.Errorf("Error not returned on invalid addition of Node: %s", err)
	} else if !strings.Contains(err.Error(), "cannot add a Node which "+
		"already exists") {
		t.Errorf("Incorrect error returned from failed AddNode: %s", err)
	}

	n := sm.nodeStates[*nid]

	if n.activity != current.WAITING {
		t.Errorf("Extant Node state has wrong activity; "+
			"Expected: %s, Recieved: %s", current.WAITING, n.activity)
	}

	if n.currentRound == nil || n.currentRound.GetRoundID() != r.GetRoundID() {
		t.Errorf("New Node has a curent round set incorrectly: "+
			"Expected: %+v; Recieved: %+v", r.GetRoundID(), n.currentRound.GetRoundID())

	}

	pollDelta := time.Now().Sub(n.lastPoll)

	if pollDelta < time.Millisecond {
		t.Errorf("timestap is too new: %v", n.lastPoll)
	}
}

//Tests a Node is retrieved correctly when in the state map
func TestStateMap_GetNode_Valid(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.ID]*State),
	}

	nid := id.NewIdFromUInt(2, id.Node, t)
	r := round.NewState_Testing(42, 0, nil, t)

	sm.nodeStates[*nid] = &State{
		activity:     current.NOT_STARTED,
		currentRound: r,
		lastPoll:     time.Now(),
	}

	n := sm.GetNode(nid)

	if n == nil {
		t.Errorf("No Node returned when Node exists")
	} else {
		if n.activity != current.NOT_STARTED {
			t.Errorf("New Node state has wrong activity; "+
				"Expected: %s, Recieved: %s", current.NOT_STARTED, n.activity)
		}

		if n.currentRound == nil || n.currentRound.GetRoundID() != r.GetRoundID() {
			t.Errorf("New Node has a curent round set incorrectly: "+
				"Expected: %+v; Recieved: %+v", r.GetRoundID(), n.currentRound.GetRoundID())

		}

		pollDelta := time.Now().Sub(n.lastPoll)

		if pollDelta < 0 || pollDelta > time.Millisecond {
			t.Errorf("timestap on poll is at the wrong time: %v", n.lastPoll)
		}
	}

}

//Tests a Node not is not returned when no Node exists
func TestStateMap_GetNode_invalid(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.ID]*State),
	}

	nid := id.NewIdFromUInt(2, id.Node, t)

	n := sm.GetNode(nid)

	if n != nil {
		t.Errorf("Nnode returned when Node does not exist")
	}
}

//Tests that len returns the correct value
func TestStateMap_Len(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for i := 0; i < 20; i++ {
		l := int(rng.Uint64() % 100)

		if i == 0 {
			l = 0
		}

		sm := &StateMap{
			nodeStates: make(map[id.ID]*State),
		}

		for j := 0; j < l; j++ {
			sm.nodeStates[*id.NewIdFromUInt(uint64(5*j+1), id.Node, t)] = &State{}
		}

		if sm.Len() != l {
			t.Errorf("Len returned a length of %v when it should be %v",
				sm.Len(), l)
		}
	}
}

// Happy path
func TestStateMap_GetNodeStates(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.ID]*State),
	}

	err := sm.AddNode(id.NewIdFromBytes([]byte("test"), t), "test", "", "", 0)
	if err != nil {
		t.Errorf("Unable to add node: %+v", err)
	}
	err = sm.AddNode(id.NewIdFromBytes([]byte("test2"), t), "test2", "", "", 0)
	if err != nil {
		t.Errorf("Unable to add node: %+v", err)
	}
	err = sm.AddNode(id.NewIdFromBytes([]byte("test3"), t), "test3", "", "", 0)
	if err != nil {
		t.Errorf("Unable to add node: %+v", err)
	}

	nodeStates := sm.GetNodeStates()
	if len(nodeStates) != 3 {
		t.Errorf("Incorrect number of nodes returned, got %d", len(nodeStates))
	}
}
