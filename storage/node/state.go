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
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/elixxir/registration/transition"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Tracks state of an individual Node in the network
type State struct {
	mux sync.RWMutex

	// Current activity as reported by the Node
	activity current.Activity

	// denotes the current status of the Node in the network
	status Status

	//nil if not in a round, otherwise holds the round the Node is in
	currentRound *round.State

	// Timestamp of the last time this Node polled
	lastPoll time.Time

	// Number of polls made by the node during the current monitoring period
	numPolls *uint64

	// Order string to be used in team configuration
	ordering string

	//holds valid state transitions
	stateMap *[][]bool

	//id of the Node
	id *id.ID

	// when a Node poll is received, this nodes polling lock is. If
	// there is no update, it is released in this endpoint, otherwise it is
	// released in the scheduling algorithm which blocks all future polls until
	// processing completes
	//FIXME: it is possible that polling lock and registration lock
	// do the same job and could conflict. reconsideration of this logic
	// may be fruitful
	pollingLock sync.Mutex
}

// Getter for numPolls
func (n *State) NumPolls() uint64 {
	return atomic.LoadUint64(n.numPolls)
}

// Increment function for numPolls
func (n *State) IncrementNumPolls() {
	atomic.AddUint64(n.numPolls, 1)
}

// Reset function for numPolls
func (n *State) ResetNumPolls() {
	atomic.SwapUint64(n.numPolls, 0)
}

// sets the Node to banned and then returns an update notification for signaling
func (n *State) Ban() (UpdateNotification, error) {
	// Get and lock n state
	n.mux.Lock()
	defer n.mux.Unlock()

	//check if the Node is already banned. do not continue if it is
	if n.status == Banned {
		return UpdateNotification{}, errors.New("cannot ban an already banned Node")
	}

	oldStatus := n.status

	//ban the Node
	n.status = Banned

	//create the update notification
	nun := UpdateNotification{
		Node:         n.id,
		FromStatus:   oldStatus,
		ToStatus:     n.status,
		FromActivity: n.activity,
		ToActivity:   n.activity,
	}

	return nun, nil
}

// updates to the passed in activity if it is different from the known activity
// returns true if the state changed and the state was it was regardless
func (n *State) Update(newActivity current.Activity) (bool, UpdateNotification, error) {
	// Get and lock n state
	n.mux.Lock()
	defer n.mux.Unlock()

	// update n poll timestamp
	n.lastPoll = time.Now()

	oldActivity := n.activity

	//if the Node is inactive, check if requirements are met to reactive it
	if n.status == Inactive {
		return n.updateInactive(newActivity)
	}

	//if the activity is the one that the Node is already in, do nothing
	if oldActivity == newActivity {
		return false, UpdateNotification{}, nil
	}

	// Check the round error state
	roundError := n.currentRound != nil && n.currentRound.GetRoundState() == states.FAILED && newActivity != current.ERROR
	if roundError {

		return false, UpdateNotification{}, errors.New("Round has failed, state cannot be updated")
	}

	//check that teh activity transition is valid
	valid := transition.Node.IsValidTransition(newActivity, oldActivity)

	if !valid {
		return false, UpdateNotification{},
			errors.Errorf("Node update from %s to %s failed, "+
				"invalid transition", oldActivity, newActivity)
	}

	// check that the state of the round the Node is assoceated with is correct
	// for the transition
	if transition.Node.NeedsRound(newActivity) == transition.Yes {

		if n.currentRound == nil {
			return false, UpdateNotification{},
				errors.Errorf("Node update from %s to %s failed, "+
					"requires the Node be assigned a round", oldActivity,
					newActivity)
		}

		if n.currentRound.GetRoundState() != transition.Node.RequiredRoundState(newActivity) {
			return false, UpdateNotification{},
				errors.Errorf("Node update from %s to %s failed, "+
					"requires the Node's be assigned a round to be in the "+
					"correct state; Assigned: %s, Expected: %s", oldActivity,
					newActivity, n.currentRound.GetRoundState(),
					transition.Node.RequiredRoundState(oldActivity))
		}
	}

	//check that the Node doesnt have a round if it shouldn't
	if transition.Node.NeedsRound(newActivity) == transition.No && n.currentRound != nil {
		return false, UpdateNotification{},
			errors.Errorf("Node update from %s to %s failed, "+
				"requires the Node not be assigned a round", oldActivity,
				newActivity)
	}

	// change the Node's activity
	n.activity = newActivity

	//build the update notification
	nun := UpdateNotification{
		Node:         n.id,
		FromStatus:   n.status,
		ToStatus:     n.status,
		FromActivity: oldActivity,
		ToActivity:   newActivity,
	}

	return true, nun, nil
}

// gets the current activity of the Node
func (n *State) GetActivity() current.Activity {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.activity
}

// Gets the status of the Node in the network
func (n *State) GetStatus() Status {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.status
}

// Gets if the Node is banned from the network
func (n *State) IsBanned() bool {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.status == Banned
}

// Designates the node as offline
func (n *State) SetInactive() {
	n.mux.RLock()
	defer n.mux.RUnlock()
	n.status = Inactive
}

// gets the timestap of the last time the Node polled
func (n *State) GetLastPoll() time.Time {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.lastPoll
}

// Returns the polling lock
func (n *State) GetPollingLock() *sync.Mutex {
	return &n.pollingLock
}

// gets the ordering string for use in team formation
func (n *State) GetOrdering() string {
	return n.ordering
}

// gets the ID of the Node
func (n *State) GetID() *id.ID {
	return n.id
}

// returns true and the round id if the Node is assigned to a round,
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

// sets the Node to not be in a round
func (n *State) ClearRound() {
	n.mux.Lock()
	defer n.mux.Unlock()
	n.currentRound = nil
}

// sets the Node's round to the passed in round unless one is already set,
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

// Handles the node update in the case of a node with an inactive state
func (n *State) updateInactive(newActivity current.Activity) (bool, UpdateNotification, error) {
	switch newActivity {
	case current.WAITING:
		oldActivity := n.activity
		n.activity = newActivity
		n.status = Active
		nun := UpdateNotification{
			Node:         n.id,
			FromStatus:   Inactive,
			ToStatus:     Active,
			FromActivity: oldActivity,
			ToActivity:   newActivity,
		}
		return true, nun, nil
	case current.ERROR:
		return false, UpdateNotification{}, nil
	default:
		return false, UpdateNotification{}, errors.Errorf("Report "+
			"for state %s rejected due to Node being inactive, Node must "+
			"activate by polling warning state", newActivity)
	}
}

func (n *State) SetLastPoll(lastPoll time.Time, t *testing.T) {
	if t == nil {
		panic("Cannot directly set node.State's last poll outside of testing")
	}
	n.lastPoll = lastPoll
}

func (n *State) SetOrdering(ordering string, t *testing.T) {
	if t == nil {
		panic("Cannot directly set node.State's ordering outside of testing")
	}
	n.ordering = ordering
}
