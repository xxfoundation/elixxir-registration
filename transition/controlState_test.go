////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package transition

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"reflect"
	"testing"
)

func TestTransitions_IsValidTransition(t *testing.T) {
	testTransition := newTransitions()
	t.Log(Node)

	// -------------- NOT_STARTED ------------------
	expectedTransition := []bool{false, true, false, true, false, false, true, false}
	receivedTransitions := make([]bool, len(expectedTransition))

	for i := uint32(0); i < uint32(current.NUM_STATES); i++ {
		//fmt.Printf("iter %d: %v\n", i, current.Activity(i))
		ok := testTransition.IsValidTransition(current.NOT_STARTED, current.Activity(i))
		receivedTransitions[i] = ok
	}

	if !reflect.DeepEqual(expectedTransition, receivedTransitions) {
		t.Errorf("State transitions for NOT_STARTED did not match expected.\n\tExpected: %v\n\tReceived: %v",
			expectedTransition, receivedTransitions)
	}

	// -------------- WAITING ------------------
	expectedTransition = []bool{false, false, false, true, false, true, true, false}

	for i := uint32(0); i < uint32(current.NUM_STATES); i++ {
		//fmt.Printf("iter %d: %v\n", i, current.Activity(i))
		ok := testTransition.IsValidTransition(current.WAITING, current.Activity(i))
		receivedTransitions[i] = ok
	}

	if !reflect.DeepEqual(expectedTransition, receivedTransitions) {
		t.Errorf("State transitions for WAITING did not match expected.\n\tExpected: %v\n\tReceived: %v",
			expectedTransition, receivedTransitions)
	}

	// -------------- PRECOMPUTING ------------------
	expectedTransition = []bool{false, true, false, false, false, false, false, false}

	for i := uint32(0); i < uint32(current.NUM_STATES); i++ {
		//fmt.Printf("iter %d: %v\n", i, current.Activity(i))
		ok := testTransition.IsValidTransition(current.PRECOMPUTING, current.Activity(i))
		receivedTransitions[i] = ok
	}

	if !reflect.DeepEqual(expectedTransition, receivedTransitions) {
		t.Errorf("State transitions for PRECOMPUTING did not match expected.\n\tExpected: %v\n\tReceived: %v",
			expectedTransition, receivedTransitions)
	}

	// -------------- REALTIME ------------------

	expectedTransition = []bool{false, false, false, true, false, false, false, false}

	for i := uint32(0); i < uint32(current.NUM_STATES); i++ {
		//fmt.Printf("iter %d: %v\n", i, current.Activity(i))
		ok := testTransition.IsValidTransition(current.REALTIME, current.Activity(i))
		receivedTransitions[i] = ok
	}

	if !reflect.DeepEqual(expectedTransition, receivedTransitions) {
		t.Errorf("State transitions for REALTIME did not match expected.\n\tExpected: %v\n\tReceived: %v",
			expectedTransition, receivedTransitions)
	}

	// -------------- COMPLETED ------------------

	expectedTransition = []bool{false, false, false, false, true, false, false, false}

	for i := uint32(0); i < uint32(current.NUM_STATES); i++ {
		//fmt.Printf("iter %d: %v\n", i, current.Activity(i))
		ok := testTransition.IsValidTransition(current.COMPLETED, current.Activity(i))
		receivedTransitions[i] = ok
	}

	if !reflect.DeepEqual(expectedTransition, receivedTransitions) {
		t.Errorf("State transitions for COMPLETED did not match expected.\n\tExpected: %v\n\tReceived: %v",
			expectedTransition, receivedTransitions)
	}

	// -------------- ERROR ------------------

	expectedTransition = []bool{true, true, true, true, true, true, false, false}

	for i := uint32(0); i < uint32(current.NUM_STATES); i++ {
		//fmt.Printf("iter %d: %v\n", i, current.Activity(i))
		ok := testTransition.IsValidTransition(current.CRASH, current.Activity(i))
		receivedTransitions[i] = ok
	}

	if !reflect.DeepEqual(expectedTransition, receivedTransitions) {
		t.Errorf("State transitions did not match expected.\n\tExpected: %v\n\tReceived: %v",
			expectedTransition, receivedTransitions)
	}
}

func TestTransitions_NeedsRound(t *testing.T) {
	testTransition := newTransitions()

	expectedNeedsRound := []int{0, 0, 1, 1, 1, 1, 2}
	receivedNeedsRound := make([]int,len(expectedNeedsRound))
	for i := uint32(0); i < uint32(current.CRASH); i++ {
		receivedNeedsRound[i] = testTransition.NeedsRound(current.Activity(i))
	}

	if !reflect.DeepEqual(expectedNeedsRound, receivedNeedsRound) {
		t.Errorf("NeedsRound did not match expected.\n\tExpected: %v\n\tReceived: %v",
			expectedNeedsRound, receivedNeedsRound)
	}
}

func TestTransitions_RequiredRoundState(t *testing.T) {
	testTransition := newTransitions()

	expectedRoundState := []states.Round{0, 0, 1, 1, 3, 3, 0}
	receivedRoundState := make([]states.Round,len(expectedRoundState))

	for i := uint32(0); i < uint32(current.CRASH); i++ {
		receivedRoundState[i] = testTransition.RequiredRoundState(current.Activity(i))
	}

	if !reflect.DeepEqual(expectedRoundState, receivedRoundState) {
		t.Errorf("NeedsRound did not match expected.\n\tExpected: %v\n\tReceived: %v",
			expectedRoundState, receivedRoundState)
	}


}