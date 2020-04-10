////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package transition

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"math"
)

const(
	No           = 0
	Yes          = 1
	Maybe  = 2
	nilRoundState = math.MaxUint32
)

var Node = newTransitions()

type Transitions [current.NUM_STATES]transitionValidation

func newTransitions() Transitions {
	t := Transitions{}
	t[current.NOT_STARTED] = NewTransitionValidation(No, nilRoundState)
	t[current.WAITING] = NewTransitionValidation(No, nilRoundState, current.NOT_STARTED, current.COMPLETED, current.ERROR)
	t[current.PRECOMPUTING] = NewTransitionValidation(Yes, states.PRECOMPUTING, current.WAITING)
	t[current.STANDBY] = NewTransitionValidation(Yes, states.PRECOMPUTING, current.PRECOMPUTING)
	t[current.REALTIME] = NewTransitionValidation(Yes, states.REALTIME, current.STANDBY)
	t[current.COMPLETED] = NewTransitionValidation(Yes, states.REALTIME, current.REALTIME)
	t[current.ERROR] = NewTransitionValidation(Maybe, nilRoundState, current.NOT_STARTED,
		current.WAITING, current.PRECOMPUTING, current.STANDBY, current.REALTIME,
		current.COMPLETED)

	return t
}

func (t Transitions)IsValidTransition(to, from current.Activity)bool{
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


func NewTransitionValidation(needsRound int, roundState states.Round, from ...current.Activity)transitionValidation{
	tv := transitionValidation{}

	tv.needsRound = needsRound
	tv.roundState = roundState

	for _, f := range from {
		tv.from[f] = true
	}

	return tv
}


