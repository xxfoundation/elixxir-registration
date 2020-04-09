package node

import (
	"errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"math"
	"sync"
	"time"
)

// Tracks state of an individual Node in the network
type State struct {
	mux sync.RWMutex

	// Current activity as reported by the Node
	activity current.Activity

	//nil if not in a round, otherwise holds the round the node is in
	currentRound *id.Round

	// Timestamp of the last time this Node polled
	lastPoll time.Time

	//holds valid state transitions
	stateMap *[][]bool
}

// updates to the passed in activity if it is different from the known activity
// returns true if the state changed and the state was it was reguardless
func (n *State) Update(newActivity current.Activity)(bool, current.Activity, error){
	// Get and lock n state
	n.mux.Lock()
	defer n.mux.Unlock()

	// update n poll timestamp
	n.lastPoll = time.Now()

	updated := false
	oldActivity := n.activity

	// change the state if the new differs from the old
	if n.activity != newActivity {

		updated = true

		n.activity = newActivity
	}

	return updated, oldActivity, nil
}

// gets the current activity of the node
func (n *State) GetActivity()current.Activity{
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.activity
}

// gets the timestap of the last time the node polled
func (n *State) GetLastPoll()time.Time{
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.lastPoll
}

// returns true and the round id if the node is assigned to a round,
// return false and Uint64Max if it is not
func (n *State) GetCurrentRound()(bool, id.Round){
	n.mux.RLock()
	defer n.mux.RUnlock()
	if n.currentRound==nil{
		return false, math.MaxUint64
	}else{
		return true, *n.currentRound
	}
}

// sets the node to not be in a round
func (n *State) ClearRound(){
	n.mux.Lock()
	defer n.mux.Unlock()
	n.currentRound = nil
}

// sets the node's round to the passed in round unless one is already set,
// in which case it errors
func (n *State) SetRound(r id.Round)error{
	n.mux.Lock()
	defer n.mux.Unlock()
	if n.currentRound!=nil{
		return errors.New("could not set the Node's round when it is " +
			"already set")
	}

	n.currentRound = &r
	return nil
}




