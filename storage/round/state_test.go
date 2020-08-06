////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"bytes"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/primitives/id"
	"math"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestState_GetLastUpdate(t *testing.T) {
	rid := id.Round(42)

	const (
		batchSize = 32
		numNodes  = 5
	)

	topology := buildMockTopology(numNodes, t)
	origTime := time.Now()
	ns := newState(rid, batchSize, 5*time.Minute, topology, origTime)

	err := ns.Update(states.PRECOMPUTING, time.Now())
	if err != nil {
		t.Errorf("Updating state failed: %v", err)
	}

	newTime := ns.GetLastUpdate()

	if origTime.After(newTime) || origTime.Equal(newTime) {
		t.Errorf("origTime was after or euqal to newTime")
	}
}

func TestNewState(t *testing.T) {
	rid := id.Round(42)

	const (
		batchSize = 32
		numNodes  = 5
	)

	topology := buildMockTopology(numNodes, t)

	ts := time.Now()

	ns := newState(rid, batchSize, 5*time.Minute, topology, ts)

	if len(ns.base.Timestamps) != int(states.NUM_STATES) {
		t.Errorf("Length of timestamps list is incorrect: "+
			"Expected: %v, Recieved: %v", numNodes, len(ns.base.Timestamps))
		t.FailNow()
	}

	expectedTimestamps := make([]uint64, states.NUM_STATES)
	expectedTimestamps[states.PENDING] = uint64(ts.Unix())

	for i := states.Round(0); i < states.NUM_STATES; i++ {
		if ns.base.Timestamps[i] != expectedTimestamps[i] {
			t.Errorf("Pending timestamp for %s is incorrect; expected: %v, :"+
				"recieved: %v", i, expectedTimestamps[i], ns.base.Timestamps[i])
		}
	}

	if len(ns.base.Topology) != numNodes {
		t.Errorf("Toplogy in pb is the wrong length: "+
			"Expected: %v, Recieved: %v", numNodes, len(ns.base.Topology))
		t.FailNow()
	}

	for i := 0; i < topology.Len(); i++ {
		strId := topology.GetNodeAtIndex(i).String()
		if bytes.Equal(ns.base.Topology[i], []byte(strId)) {
			t.Errorf("Topology string on index %v is incorrect"+
				"Expected: %s, Recieved: %s", i, strId, ns.base.Topology[i])
		}
	}

	if !reflect.DeepEqual(topology, ns.topology) {
		t.Errorf("Topology in round not the same as passed in")
	}

	if ns.base.BatchSize != batchSize {
		t.Errorf("BatchSize in pb is incorrect; "+
			"Expected: %v, Recieved: %v", batchSize, ns.base.BatchSize)
	}

	if ns.base.ID != uint64(rid) {
		t.Errorf("round ID in pb is incorrect; "+
			"Expected: %v, Recived: %v", rid, ns.base.ID)
	}

	if ns.base.UpdateID != math.MaxUint64 {
		t.Errorf("update ID in pb is incorrect; "+
			"Expected: %v, Recived: %v", uint64(math.MaxUint64),
			ns.base.UpdateID)
	}

	if ns.state != states.PENDING {
		t.Errorf("State of round is incorrect; "+
			"Expected: %s, Recived: %s", states.PENDING, ns.state)
	}

	if ns.readyForTransition != 0 {
		t.Errorf("readyForTransmission is incorrect; "+
			"Expected: %v, Recived: %v", 0, ns.readyForTransition)
	}

}

// tests all rollover and non rollover transitions
func TestState_NodeIsReadyForTransition(t *testing.T) {
	rid := id.Round(42)

	const (
		batchSize = 32
		numNodes  = 5
	)

	topology := buildMockTopology(numNodes, t)

	ts := time.Now()

	ns := newState(rid, batchSize, 5*time.Minute, topology, ts)

	if ns.readyForTransition != 0 {
		t.Errorf("readyForTransmission is incorrect; "+
			"Expected: %v, Recived: %v", 0, ns.readyForTransition)
	}

	//test all non roll over transitions
	for i := 0; i < numNodes-1; i++ {
		ready := ns.NodeIsReadyForTransition()

		if ready {
			t.Errorf("state should not be ready for transition on the "+
				"%vth node", i)
		}

		if int(ns.readyForTransition) != i+1 {
			t.Errorf("Ready for transition counter not correct; "+
				"Expected: %v, recieved: %v", i+1, ns.readyForTransition)
		}
	}
}

//test the state update increments properly when given a valid input
func TestState_Update_Forward(t *testing.T) {
	rid := id.Round(42)

	const (
		batchSize = 32
		numNodes  = 5
	)

	topology := buildMockTopology(numNodes, t)

	ts := time.Now()

	ns := newState(rid, batchSize, 5*time.Minute, topology, ts)

	for i := states.PRECOMPUTING; i < states.NUM_STATES; i++ {
		time.Sleep(1 * time.Millisecond)
		ts = time.Now()
		err := ns.Update(i, ts)
		if err != nil {
			t.Errorf("state update failed on valid transition to %s",
				i)
		}
		if ns.state != i {
			t.Errorf("Transition to state %s failed, at state %s", i,
				ns.state)
		}

		if ns.base.Timestamps[i] != uint64(ts.UnixNano()) {
			t.Errorf("Timestamp stored is incorrect. "+
				"Stored: %v, Expected: %v", ns.base.Timestamps[i], uint64(ts.Unix()))
		}
	}
}

//test the state update errors properly when set the the same state it is at
func TestState_Update_Same(t *testing.T) {
	rid := id.Round(42)

	const (
		batchSize = 32
		numNodes  = 5
	)

	topology := buildMockTopology(numNodes, t)

	ts := time.Now()

	ns := newState(rid, batchSize, 5*time.Minute, topology, ts)

	for i := states.PENDING; i < states.NUM_STATES; i++ {
		ns.state = i
		ns.base.Timestamps[i] = math.MaxUint64
		time.Sleep(1 * time.Millisecond)
		ts = time.Now()
		err := ns.Update(i, ts)
		if err == nil {
			t.Errorf("state update succeded on invalid transition "+
				"to %s from %s", i, i)
		} else if !strings.Contains(err.Error(), "round state must "+
			"always update to a greater state") {
			t.Errorf("state update failed with incorrect error: %s",
				err)
		}

		if ns.state != i {
			t.Errorf("State incorrect after lateral transition for state "+
				"%s resulted in final state of %s", i, ns.state)
		}

		if ns.base.Timestamps[i] != math.MaxUint64 {
			t.Errorf("Timestamp edited on failed update"+
				"Stored: %v, Expected: %v", ns.base.Timestamps[i],
				uint64(math.MaxUint64))
		}
	}
}

// test the state update errors properly when set to a state less than the
// current one
func TestState_Update_Reverse(t *testing.T) {
	rid := id.Round(42)

	const (
		batchSize = 32
		numNodes  = 5
	)

	topology := buildMockTopology(numNodes, t)

	ts := time.Now()

	ns := newState(rid, batchSize, 5*time.Minute, topology, ts)

	for i := states.PRECOMPUTING; i < states.NUM_STATES; i++ {
		ns.state = i
		ns.base.Timestamps[i] = math.MaxUint64
		time.Sleep(1 * time.Millisecond)
		ts = time.Now()
		err := ns.Update(i-1, ts)
		if err == nil {
			t.Errorf("state update succeded on invalid transition "+
				"to %s from %s", i-1, i)
		} else if !strings.Contains(err.Error(), "round state must "+
			"always update to a greater state") {
			t.Errorf("state update failed with incorrect error: %s",
				err)
		}

		if ns.state != i {
			t.Errorf("State incorrect after reverse transition to state "+
				"%s from %s resulting in final state of %s", i, i-1, ns.state)
		}

		if ns.base.Timestamps[i] != math.MaxUint64 {
			t.Errorf("Timestamp edited on failed update"+
				"Stored: %v, Expected: %v", ns.base.Timestamps[i],
				uint64(math.MaxUint64))
		}
	}
}

func TestState_BuildRoundInfo(t *testing.T) {
	rid := id.Round(42)

	const (
		batchSize = 32
		numNodes  = 5
	)

	topology := buildMockTopology(numNodes, t)

	ts := time.Now()

	ns := newState(rid, batchSize, 5*time.Minute, topology, ts)

	ns.state = states.FAILED

	ri := ns.BuildRoundInfo()

	if len(ri.Timestamps) != int(states.NUM_STATES) {
		t.Errorf("Length of timestamps list is incorrect: "+
			"Expected: %v, Recieved: %v", numNodes, len(ri.Timestamps))
		t.FailNow()
	}

	expectedTimestamps := make([]uint64, states.NUM_STATES)
	expectedTimestamps[states.PENDING] = uint64(ts.Unix())

	for i := states.Round(0); i < states.NUM_STATES; i++ {
		if ri.Timestamps[i] != expectedTimestamps[i] {
			t.Errorf("Pending timestamp for %s is incorrect; expected: %v, :"+
				"recieved: %v", i, expectedTimestamps[i], ri.Timestamps[i])
		}
	}

	if len(ns.base.Topology) != numNodes {
		t.Errorf("Toplogy in pb is the wrong length: "+
			"Expected: %v, Recieved: %v", numNodes, len(ri.Topology))
		t.FailNow()
	}

	for i := 0; i < topology.Len(); i++ {
		strId := topology.GetNodeAtIndex(i).String()
		if bytes.Equal(ri.Topology[i], []byte(strId)) {
			t.Errorf("Topology string on index %v is incorrect"+
				"Expected: %s, Recieved: %s", i, strId, ri.Topology[i])
		}
	}

	if ri.UpdateID != 0 {
		t.Errorf("update ID is incorrect; Expected: %v, Recieved: %v",
			0, ri.UpdateID)
	}

	if ri.ID != uint64(rid) {
		t.Errorf("Round ID is incorrect; Expected: %v, Recieved: %v",
			rid, ri.ID)
	}

	if ri.BatchSize != batchSize {
		t.Errorf("Batchsize is incorrect; Expected: %v, Recieved: %v",
			batchSize, ri.BatchSize)
	}

	if ri.State != uint32(states.FAILED) {
		t.Errorf("State is incorrect; Expected: %v, Recieved: %v",
			states.FAILED, ri.State)
	}
}

//tests that GetRoundState returns the correct state
func TestState_GetRoundState(t *testing.T) {
	for i := states.Round(0); i < states.NUM_STATES; i++ {
		rs := State{state: i}

		s := rs.GetRoundState()

		if s != i {
			t.Errorf("GetRoundState returned the incorrect state;"+
				"Expected: %s, Recieved: %s", i, s)
		}
	}
}

//tests that GetTopology returns the correct topology
func TestState_GetTopology(t *testing.T) {

	const (
		numNodes = 5
	)

	topology := buildMockTopology(numNodes, t)

	rs := State{topology: topology}

	if !reflect.DeepEqual(topology, rs.GetTopology()) {
		t.Errorf("retruned topology did not match passed topology")
	}
}
