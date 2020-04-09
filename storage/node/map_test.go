package node

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"strings"
	"testing"
	"time"
)

//tests that newStateMap is correct
func TestNewStateMap(t *testing.T) {
	sm := NewStateMap()

	if sm.nodeStates==nil{
		t.Errorf("Internal map not initilized")
	}
}

//Tests a node is added correctly to the state map when it is
func TestStateMap_AddNode_Happy(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.Node]*State),
	}

	nid := id.NewNodeFromUInt(2, t)

	err := sm.AddNode(nid)

	if err!=nil{
		t.Errorf("Error returned on valid addition of node: %s", err)
	}

	n := sm.nodeStates[*nid]

	if n.activity != current.NOT_STARTED{
		t.Errorf("New node state has wrong activity; "+
			"Expected: %s, Recieved: %s", current.NOT_STARTED, n.activity)
	}

	if n.currentRound!=nil{
		t.Errorf("New node has a curent round set incorrectly")
	}

	pollDelta := time.Now().Sub(n.lastPoll)

	if pollDelta <0 || pollDelta>time.Millisecond{
		t.Errorf("timestap on poll is at the wrong time: %v", n.lastPoll)
	}
}

//Tests a node is added correctly to the state map when it is
func TestStateMap_AddNode_Invalid(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.Node]*State),
	}

	nid := id.NewNodeFromUInt(2, t)
	rid := id.Round(42)

	sm.nodeStates[*nid] = &State{
		activity:     current.WAITING,
		currentRound: &rid,
		lastPoll:     time.Now(),
	}

	time.Sleep(1*time.Millisecond)

	err := sm.AddNode(nid)

	if err==nil{
		t.Errorf("Error not returned on invalid addition of node: %s", err)
	}else if ! strings.Contains(err.Error(),"cannot add a node which "+
		"already exists"){
		t.Errorf("Incorrect error returned from failed AddNode: %s", err)
	}

	n := sm.nodeStates[*nid]

	if n.activity != current.WAITING{
		t.Errorf("Extant node state has wrong activity; "+
			"Expected: %s, Recieved: %s", current.WAITING, n.activity)
	}

	if n.currentRound==nil || *n.currentRound!=rid{
		t.Errorf("New node has a curent round set incorrectly: " +
			"Expected: %+v; Recieved: %+v", &rid, n.currentRound)

	}

	pollDelta := time.Now().Sub(n.lastPoll)

	if pollDelta < time.Millisecond{
		t.Errorf("timestap is too new: %v", n.lastPoll)
	}
}

//Tests a node is retrieved correctly when in the state map
func TestStateMap_GetNode_Valid(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.Node]*State),
	}

	nid := id.NewNodeFromUInt(2, t)
	rid := id.Round(42)

	sm.nodeStates[*nid] = &State{
		activity:     current.NOT_STARTED,
		currentRound: &rid,
		lastPoll:     time.Now(),
	}

	n := sm.GetNode(nid)

	if n==nil{
		t.Errorf("No node returned when node exists")
	}else{
		if n.activity != current.NOT_STARTED{
			t.Errorf("New node state has wrong activity; "+
				"Expected: %s, Recieved: %s", current.NOT_STARTED, n.activity)
		}

		if n.currentRound==nil || *n.currentRound!=rid{
			t.Errorf("New node has a curent round set incorrectly: " +
				"Expected: %+v; Recieved: %+v", &rid, n.currentRound)

		}

		pollDelta := time.Now().Sub(n.lastPoll)

		if pollDelta <0 || pollDelta>time.Millisecond{
			t.Errorf("timestap on poll is at the wrong time: %v", n.lastPoll)
		}
	}

}

//Tests a not is not returned when no node exists
func TestStateMap_GetNode_invalid(t *testing.T) {
	sm := &StateMap{
		nodeStates: make(map[id.Node]*State),
	}

	nid := id.NewNodeFromUInt(2, t)


	n := sm.GetNode(nid)

	if n!=nil{
		t.Errorf("Nnode returned when node does not exist")
	}
}