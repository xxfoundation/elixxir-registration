package node

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage/round"
	"math/rand"
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

	err := sm.AddNode(nid, "")

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
	r := round.NewState_Testing(42, 0, t)

	sm.nodeStates[*nid] = &State{
		activity:     current.WAITING,
		currentRound: r,
		lastPoll:     time.Now(),
	}

	time.Sleep(1*time.Millisecond)

	err := sm.AddNode(nid, "")

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

	if n.currentRound==nil || n.currentRound.GetRoundID()!=r.GetRoundID(){
		t.Errorf("New node has a curent round set incorrectly: " +
			"Expected: %+v; Recieved: %+v", r.GetRoundID(), n.currentRound.GetRoundID())

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
	r := round.NewState_Testing(42, 0, t)

	sm.nodeStates[*nid] = &State{
		activity:     current.NOT_STARTED,
		currentRound: r,
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

		if n.currentRound==nil || n.currentRound.GetRoundID()!=r.GetRoundID(){
			t.Errorf("New node has a curent round set incorrectly: " +
				"Expected: %+v; Recieved: %+v", r.GetRoundID(), n.currentRound.GetRoundID())

		}

		pollDelta := time.Now().Sub(n.lastPoll)

		if pollDelta <0 || pollDelta>time.Millisecond{
			t.Errorf("timestap on poll is at the wrong time: %v", n.lastPoll)
		}
	}

}

//Tests a node not is not returned when no node exists
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

//Tests that len returns the correct value
func TestStateMap_Len(t *testing.T) {
	rng := rand.New(rand.NewSource(42))

	for i:=0;i<20;i++{
		l := int(rng.Uint64()%100)

		if i==0{
			l=0
		}

		sm := &StateMap{
			nodeStates: make(map[id.Node]*State),
		}

		for j:=0;j<l;j++{
			sm.nodeStates[*id.NewNodeFromUInt(uint64(5*j+1), t)] = &State{}
		}

		if sm.Len()!=l{
			t.Errorf("Len returned a length of %v when it should be %v",
				sm.Len(), l)
		}
	}
}