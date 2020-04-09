package node

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"math"
	"strings"
	"testing"
	"time"
)

// tests that State update functions properly when the state it is updated
// to is not the one it is at
func TestNodeState_Update_Same(t *testing.T) {
	ns := State{
		activity:     current.WAITING,
		lastPoll:     time.Now(),
	}

	time.Sleep(10*time.Millisecond)

	before := time.Now()

	updated, old := ns.Update(current.WAITING)
	timeDelta := ns.lastPoll.Sub(before)
	if timeDelta>(1*time.Millisecond) || timeDelta<0{
		t.Errorf("Time recorded is not between 0 and 1 ms from " +
			"checkpoint: %s", timeDelta)
	}

	if updated==true{
		t.Errorf("Node state should not have updated")
	}

	if old!=current.WAITING{
		t.Errorf("Node state returned the wrong old state")
	}

	if ns.activity!=current.WAITING{
		t.Errorf("Internal node activity is not correct: " +
			"Expected: %s, Recieved: %s", current.WAITING, ns.activity)
	}
}

// tests that State update functions properly when the state it is updated
// to is not the one it is not at
func TestNodeState_Update_Different(t *testing.T) {
	ns := State{
		activity:     current.WAITING,
		lastPoll:     time.Now(),
	}

	time.Sleep(10*time.Millisecond)

	before := time.Now()

	updated, old := ns.Update(current.STANDBY)
	timeDelta := ns.lastPoll.Sub(before)
	if timeDelta>(1*time.Millisecond) || timeDelta<0{
		t.Errorf("Time recorded is not between 0 and 1 ms from " +
			"checkpoint: %s", timeDelta)
	}

	if updated==false{
		t.Errorf("Node state should have updated")
	}

	if old!=current.WAITING{
		t.Errorf("Node state returned the wrong old state")
	}

	if ns.activity!=current.STANDBY{
		t.Errorf("Internal node activity is not correct: " +
			"Expected: %s, Recieved: %s", current.STANDBY, ns.activity)
	}
}

//tests that GetActivity returns the correct activity
func TestNodeState_GetActivity(t *testing.T) {
	for i:=0;i<10;i++{
		ns := State{
			activity: current.Activity(i),
		}

		a := ns.GetActivity()

		if a!=current.Activity(i){
			t.Errorf("returned curent activity not as set" +
				"Expected: %v, Recieved: %v", a, i)
		}
	}
}

//tests that GetActivity returns the correct activity
func TestNodeState_GetLastPoll (t *testing.T) {
	ns := State{}
	for i:=0;i<10;i++{
		before := time.Now()
		ns.lastPoll = before
		lp := ns.GetLastPoll()

		if lp.Sub(before)!=0{
			t.Errorf("Last Poll returned the wrong datetime")
		}
	}
}

//tests that GetActivity returns the correct activity
func TestNodeState_GetCurrentRound_Set (t *testing.T) {
	r := id.Round(42)
	ns := State{
		currentRound: &r,
	}

	success, rnd := ns.GetCurrentRound()

	if !success{
		t.Errorf("No round is set when one should be")
	}

	if rnd!=r{
		t.Errorf("Returned round is not correct: " +
			"Expected: %v, Recieved: %v", r, rnd)
	}
}

//tests that GetActivity returns the correct activity
func TestNodeState_GetCurrentRound_NotSet (t *testing.T) {
	ns := State{
	}

	success, rnd := ns.GetCurrentRound()

	if success{
		t.Errorf("round returned when none is set")
	}

	if rnd!=math.MaxUint64{
		t.Errorf("Returned round is not error valuve: " +
			"Expected: %v, Recieved: %v", uint64(math.MaxUint64), rnd)
	}
}

//tests that clear round sets the tracked roundID to nil
func TestNodeState_ClearRound(t *testing.T) {
	rid := id.Round(420)

	ns := State{
		currentRound: &rid,
	}

	ns.ClearRound()

	if ns.currentRound!=nil{
		t.Errorf("The curent round was not nilled")
	}
}

//tests that clear round sets the tracked roundID to nil
func TestNodeState_SetRound_Valid(t *testing.T) {
	rid := id.Round(42)

	ns := State{
		currentRound: nil,
	}

	err := ns.SetRound(rid)

	if err!=nil{
		t.Errorf("SetRound returned an error which it should be " +
			"sucesfull: %s", err)
	}

	if *ns.currentRound!=rid{
		t.Errorf("Round not updated to the correct value; "+
			"Expected: %v, Recieved: %v", rid, *ns.currentRound)
	}
}

//tests that clear round does not set the tracked roundID errors when one is set
func TestNodeState_SetRound_Invalid(t *testing.T) {
	rid := id.Round(42)
	storedRid := id.Round(69)

	ns := State{
		currentRound: &storedRid,
	}

	err := ns.SetRound(rid)

	if err==nil{
		t.Errorf("SetRound did not an error which it should have failed")
	}else if ! strings.Contains(err.Error(),"could not set the Node's " +
		"round when it is already set"){
		t.Errorf("Incorrect error returned from failed SetRound: %s", err)
	}

	if *ns.currentRound!=storedRid{
		t.Errorf("Round not updated to the correct value; "+
			"Expected: %v, Recieved: %v", rid, *ns.currentRound)
	}
}