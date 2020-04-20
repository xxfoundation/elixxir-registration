////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/elixxir/registration/transition"
	"sync"
	"time"
)

// Tracks state of an individual Node in the network
type State struct {
	mux sync.RWMutex

	// Current activity as reported by the Node
	activity current.Activity

	//nil if not in a round, otherwise holds the round the node is in
	currentRound *round.State

	// Timestamp of the last time this Node polled
	lastPoll time.Time

	// Order string to be used in team configuration
	ordering string

	//holds valid state transitions
	stateMap *[][]bool

	//id of the node
	id *id.Node
}

// updates to the passed in activity if it is different from the known activity
// returns true if the state changed and the state was it was reguardless
func (n *State) Update(newActivity current.Activity) (bool, current.Activity, error) {
	// Get and lock n state
	n.mux.Lock()
	defer n.mux.Unlock()

	// update n poll timestamp
	n.lastPoll = time.Now()

	oldActivity := n.activity

	//if the activity is the one that the node is already in, do nothing
	if oldActivity == newActivity {
		return false, oldActivity, nil
	}

	//check that teh activity transition is valid
	valid := transition.Node.IsValidTransition(newActivity, oldActivity)

	if !valid {
		return false, oldActivity,
			errors.Errorf("node update from %s to %s failed, "+
				"invalid transition", oldActivity, newActivity)
	}

	// check that the state of the round the node is assoceated with is correct
	// for the transition
	if transition.Node.NeedsRound(newActivity) == transition.Yes {
		if n.currentRound == nil {
			return false, oldActivity,
				errors.Errorf("node update from %s to %s failed, "+
					"requires the node be assigned a round", oldActivity,
					newActivity)
		}

		if n.currentRound.GetRoundState() != transition.Node.RequiredRoundState(newActivity) {
			return false, oldActivity,
				errors.Errorf("node update from %s to %s failed, "+
					"requires the node's be assigned a round to be in the "+
					"correct state; Assigned: %s, Expected: %s", oldActivity,
					newActivity, n.currentRound.GetRoundState(),
					transition.Node.RequiredRoundState(oldActivity))
		}
	}

	//check that the node doesnt have a round if it shouldn't
	if transition.Node.NeedsRound(newActivity) == transition.No && n.currentRound != nil {
		return false, oldActivity,
			errors.Errorf("node update from %s to %s failed, "+
				"requires the node not be assigned a round", oldActivity,
				newActivity)
	}

	// change the node's activity
	n.activity = newActivity

	return true, oldActivity, nil
}

// gets the current activity of the node
func (n *State) GetActivity() current.Activity {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.activity
}

// gets the timestap of the last time the node polled
func (n *State) GetLastPoll() time.Time {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.lastPoll
}

// gets the ordering string for use in team formation
func (n *State) GetOrdering() string {
	return n.ordering
}

// gets the ID of the node
func (n *State) GetID() *id.Node {
	return n.id
}

// returns true and the round id if the node is assigned to a round,
// return false and nil if it is not
func (n *State) GetCurrentRound() (bool, *round.State) {
	n.mux.RLock()
	defer n.mux.RUnlock()
	if n.currentRound == nil {
		return false, nil
	} else {
		return true, n.currentRound
	}
}

// sets the node to not be in a round
func (n *State) ClearRound() {
	n.mux.Lock()
	defer n.mux.Unlock()
	n.currentRound = nil
}

// sets the node's round to the passed in round unless one is already set,
// in which case it errors
func (n *State) SetRound(r *round.State) error {
	n.mux.Lock()
	defer n.mux.Unlock()
	if n.currentRound != nil {
		return errors.New("could not set the Node's round when it is " +
			"already set")
	}

	n.currentRound = r
	return nil
}
