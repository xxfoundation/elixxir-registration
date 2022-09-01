////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package transition

import (
	"fmt"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"math"
)

// ControlState.go contains the state transition information for nodes.

const (
	No            = 0
	Yes           = 1
	Maybe         = 2
	nilRoundState = math.MaxUint32
)

// Node is a global variable used as bookkeping for state transition information
var Node = newTransitions()

type Transitions [current.NUM_STATES]transitionValidation

// newTransition creates a transition table containing necessary information
// on state transitions
func newTransitions() Transitions {
	t := Transitions{}
	t[current.NOT_STARTED] = NewTransitionValidation(No, nil)
	t[current.WAITING] = NewTransitionValidation(No, nil, current.NOT_STARTED, current.COMPLETED, current.ERROR)
	t[current.PRECOMPUTING] = NewTransitionValidation(Yes, []states.Round{states.PRECOMPUTING}, current.WAITING)
	t[current.STANDBY] = NewTransitionValidation(Yes, []states.Round{states.PRECOMPUTING}, current.WAITING, current.PRECOMPUTING)
	t[current.REALTIME] = NewTransitionValidation(Yes, []states.Round{states.QUEUED, states.REALTIME}, current.STANDBY)
	t[current.COMPLETED] = NewTransitionValidation(Yes, []states.Round{states.REALTIME}, current.REALTIME)
	t[current.ERROR] = NewTransitionValidation(Maybe, nil, current.NOT_STARTED,
		current.WAITING, current.PRECOMPUTING, current.STANDBY, current.REALTIME,
		current.COMPLETED)

	return t
}

// IsValidTransition checks the transitionValidation to see if
//  the attempted transition is valid
func (t Transitions) IsValidTransition(to, from current.Activity) bool {
	return t[to].from[from]
}

// NeedsRound checks if the state being transitioned to
//  will need round updates
func (t Transitions) NeedsRound(to current.Activity) int {
	return t[to].needsRound
}

// Checks if the passed round state is valid for the transition
func (t Transitions) IsValidRoundState(to current.Activity, st states.Round) bool {
	for _, s := range t[to].roundState {
		if st == s {
			return true
		}
	}
	return false
}

// returns a string describing valid transitions for error messages
func (t Transitions) GetValidRoundStateStrings(to current.Activity) string {
	if to > current.NUM_STATES || to < 0 {
		return "INVALID STATE"
	}

	if len(t[to].roundState) == 0 {
		return "NO VALID TRANSITIONS"
	}

	var rtnStr string

	for i := 0; i < len(t[to].roundState)-1; i++ {
		rtnStr = fmt.Sprintf("%s, ", t[to].roundState[i])
	}

	rtnStr += fmt.Sprintf("%s", t[to].roundState[len(t[to].roundState)-1])
	return rtnStr
}

// Transitional information used for each state
type transitionValidation struct {
	from       [current.NUM_STATES]bool
	needsRound int
	roundState []states.Round
}

// NewTransitionValidation sets the from attribute,
//  denoting whether going from that to the objects current state
//  is valid
func NewTransitionValidation(needsRound int, roundState []states.Round, from ...current.Activity) transitionValidation {
	tv := transitionValidation{}

	tv.needsRound = needsRound
	tv.roundState = roundState

	for _, f := range from {
		tv.from[f] = true
	}

	return tv
}
