////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package transition

import (
	"fmt"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
)

const(
	No           = 0
	Yes          = 1
	DoesntMatter = 2
)

var Node = newTransitions()

type Transitions [current.NUM_STATES]*transitionValidation

func newTransitions() Transitions {
	t := Transitions{}
	for i := current.Activity(0); i < current.NUM_STATES; i++ {
		t[current.Activity(i)] = &transitionValidation{}
	}
	t[current.NOT_STARTED].setStateTransition()
	t[current.WAITING].setStateTransition(current.NOT_STARTED, current.COMPLETED, current.ERROR)
	t[current.PRECOMPUTING].setStateTransition(current.WAITING)

	t[current.REALTIME].setStateTransition(current.STANDBY)

	t[current.COMPLETED].setStateTransition(current.REALTIME)
	t[current.ERROR].setStateTransition(current.NOT_STARTED, current.WAITING,
		current.PRECOMPUTING, current.STANDBY, current.REALTIME,
		current.COMPLETED)

	t[current.NOT_STARTED].needsRound = No
	t[current.WAITING].needsRound = No
	t[current.PRECOMPUTING].needsRound = Yes
	t[current.PRECOMPUTING].roundState = states.PRECOMPUTING
	t[current.STANDBY].needsRound = Yes
	t[current.STANDBY].roundState = states.PRECOMPUTING
	t[current.REALTIME].needsRound = Yes
	t[current.REALTIME].roundState = states.REALTIME
	t[current.COMPLETED].needsRound = Yes
	t[current.COMPLETED].roundState = states.REALTIME
	t[current.ERROR].needsRound = DoesntMatter
	return t
}

func (t Transitions)IsValidTransition(from, to current.Activity)bool{
	//fmt.Println("from ", from, " to ", to)
	return t[to].from[from]
}

func (t Transitions)NeedsRound(to current.Activity)int{
	return t[to].needsRound
}

func (t Transitions)RequiredRoundState(to current.Activity)states.Round{
	return t[to].roundState
}

type transitionValidation struct{
	from       [current.NUM_STATES]bool
	needsRound int
	roundState states.Round
}

// adds a state transition from the state object
func (tv *transitionValidation) setStateTransition(from ...current.Activity) {
	for _, f := range from {
		tv.from[f] = true
	}
	fmt.Printf("tv.from: %v\n", tv.from)

}


