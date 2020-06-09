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
	t[current.NOT_STARTED] = NewTransitionValidation(No, nilRoundState)
	t[current.WAITING] = NewTransitionValidation(No, nilRoundState, current.NOT_STARTED, current.COMPLETED, current.ERROR)
	t[current.PRECOMPUTING] = NewTransitionValidation(Yes, states.PRECOMPUTING, current.WAITING)
	t[current.STANDBY] = NewTransitionValidation(Yes, states.PRECOMPUTING, current.PRECOMPUTING)
	t[current.REALTIME] = NewTransitionValidation(Yes, states.QUEUED, current.STANDBY)
	t[current.COMPLETED] = NewTransitionValidation(Yes, states.REALTIME, current.REALTIME)
	t[current.ERROR] = NewTransitionValidation(Maybe, nilRoundState, current.NOT_STARTED,
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

// RequiredRoundState looks up the required round needed prior to transition
func (t Transitions) RequiredRoundState(to current.Activity) states.Round {
	return t[to].roundState
}

// Transitional information used for each state
type transitionValidation struct {
	from       [current.NUM_STATES]bool
	needsRound int
	roundState states.Round
}

// NewTransitionValidation sets the from attribute,
//  denoting whether going from that to the objects current state
//  is valid
func NewTransitionValidation(needsRound int, roundState states.Round, from ...current.Activity) transitionValidation {
	tv := transitionValidation{}

	tv.needsRound = needsRound
	tv.roundState = roundState

	for _, f := range from {
		tv.from[f] = true
	}

	return tv
}
