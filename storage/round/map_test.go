////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

//tests that newStateMap is correct
func TestNewStateMap(t *testing.T) {
	sm := NewStateMap()

	if sm.rounds == nil {
		t.Errorf("Internal map not initilized")
	}
}

// Tests a round is added correctly to the state map when the round does not
// exist
func TestStateMap_AddRound_Happy(t *testing.T) {
	sm := &StateMap{
		rounds: make(map[id.Round]*State),
	}

	rid := id.Round(2)

	const numNodes = 5

	rRtn, err := sm.AddRound(rid, 32, 8, 5*time.Minute, buildMockTopology(numNodes, t))

	if err != nil {
		t.Errorf("Error returned on valid addition of node: %s", err)
	}

	r := sm.rounds[rid]

	if r == nil {
		t.Errorf("round not returned when lookup is valid")
		t.FailNow()
	}

	if rRtn.GetRoundID() != rid {
		t.Errorf("round from lookup returned with wrong id")
	}

	if r.GetRoundID() != rid {
		t.Errorf("round from lookup returned with wrong id")
	}
}

// Tests a round is not added correctly to the state map when the round
// already exists
func TestStateMap_AddNode_Invalid(t *testing.T) {
	sm := &StateMap{
		rounds: make(map[id.Round]*State),
	}

	rid := id.Round(2)

	const numNodes = 5

	sm.rounds[rid] = &State{state: states.FAILED}

	rRtn, err := sm.AddRound(rid, 32, 8, 5*time.Minute, buildMockTopology(numNodes, t))

	if err == nil {
		t.Errorf("Error not returned on invalid addition of node: %s", err)
	}

	if rRtn != nil {
		t.Errorf("round returned when none create")
	}

	if sm.rounds[rid].state != states.FAILED {
		t.Errorf("the state of the round was overweritten")
	}
}

//Tests a node is retrieved correctly when in the state map
func TestStateMap_GetRound_Valid(t *testing.T) {
	sm := &StateMap{
		rounds: make(map[id.Round]*State),
	}
	rid := id.Round(2)
	sm.rounds[rid] = &State{}

	r := sm.GetRound(rid)

	if r == nil {
		t.Errorf("Round not retrieved when valid")
	}

}

//Tests a not is not returned when no node exists
func TestStateMap_GetNode_invalid(t *testing.T) {
	sm := &StateMap{
		rounds: make(map[id.Round]*State),
	}
	rid := id.Round(2)

	r := sm.GetRound(rid)

	if r != nil {
		t.Errorf("Round retrieved when invalid")
	}
}

// Happy path
func TestStateMap_DeleteRound(t *testing.T) {
	sm := &StateMap{
		rounds: make(map[id.Round]*State),
	}
	rid := id.Round(2)
	sm.rounds[rid] = &State{}

	sm.DeleteRound(rid)

	// Check that the round is empty upon deletion
	r := sm.GetRound(rid)
	if r != nil {
		t.Errorf("Round retrieved after deletion from map")
	}

}

func buildMockTopology(numNodes int, t *testing.T) *connect.Circuit {
	nodeLst := make([]*id.ID, numNodes)
	for i := 0; i < numNodes; i++ {
		nid := id.NewIdFromUInt(uint64(i+1), id.Node, t)
		nodeLst[i] = nid
	}
	return connect.NewCircuit(nodeLst)
}
