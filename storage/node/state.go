////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"bytes"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/crypto/nike"
	"gitlab.com/elixxir/crypto/nike/ecdh"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/elixxir/registration/transition"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const ipUpdateTimeout = 30 * time.Minute

// Enumeration of connectivity statuses for a node
const (
	PortUnknown uint32 = iota
	PortVerifying
	PortSuccessful
	NodePortFailed
	GatewayPortFailed
	PortFailed
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

	// Timestamp of the last time this Node produced an update
	lastUpdate time.Time

	// Timestamp of the last time node has been updated internally
	// within the node metric tracker
	lastActive time.Time

	// Number of polls made by the node during the current monitoring period
	numPolls *uint64

	// Order string to be used in team configuration
	ordering string

	//holds valid state transitions
	stateMap *[][]bool

	//id of the Node
	id *id.ID

	//Application ID of the node
	//used primarily for logging
	applicationID uint64

	// Address of node
	nodeAddress      string
	lastNodeUpdateTS time.Time

	// Address of gateway
	gatewayAddress      string
	lastGatewayUpdateTS time.Time

	// when a Node poll is received, this nodes polling lock is. If
	// there is no update, it is released in this endpoint, otherwise it is
	// released in the scheduling algorithm which blocks all future polls until
	// processing completes
	//FIXME: it is possible that polling lock and registration lock
	// do the same job and could conflict. reconsideration of this logic
	// may be fruitful
	pollingLock sync.Mutex

	// Status of node's connectivity, i.e. whether the node
	// has port forwarding
	connectivity *uint32

	ed25519 nike.PublicKey
}

// Increment function for numPolls
func (n *State) IncrementNumPolls() {
	atomic.AddUint64(n.numPolls, 1)
}

// Returns the current value of numPolls and then resets numPolls to zero
func (n *State) GetAndResetNumPolls() uint64 {
	return atomic.SwapUint64(n.numPolls, 0)
}

func (n *State) SetNumPollsTesting(num int, x interface{}) {
	// Ensure that this function is only run in testing environments
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B:
		break
	default:
		panic("SetNumPollsTesting() can only be used for testing.")
	}

	atomic.SwapUint64(n.numPolls, uint64(num))

}

// Returns the current value of numPolls and then resets numPolls to zero
func (n *State) GetNumPolls() uint64 {
	return atomic.LoadUint64(n.numPolls)
}

// Returns the current value of numPolls and then resets numPolls to zero
func (n *State) GetAppID() uint64 {
	return n.applicationID
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

	// If the Node's round has failed, force an error transition
	if n.currentRound != nil && n.currentRound.GetRoundState() == states.FAILED && newActivity != current.ERROR {
		newActivity = current.ERROR
	}

	//if the activity is the one that the Node is already in, do nothing
	if oldActivity == newActivity {
		return false, UpdateNotification{}, nil
	}

	//check that teh activity transition is valid
	valid := transition.Node.IsValidTransition(newActivity, oldActivity)

	if !valid {
		return false, UpdateNotification{},
			errors.Errorf("Node update from %s to %s failed, "+
				"invalid transition", oldActivity, newActivity)
	}

	// check that the state of the round the Node is associated with is correct
	// for the transition
	if transition.Node.NeedsRound(newActivity) == transition.Yes {

		if n.currentRound == nil {
			return false, UpdateNotification{},
				errors.Errorf("Node update from %s to %s failed, "+
					"requires the Node be assigned a round", oldActivity,
					newActivity)
		}

		if !transition.Node.IsValidRoundState(newActivity, n.currentRound.GetRoundState()) {

			return false, UpdateNotification{},
				errors.Errorf("Node update from %s to %s failed, "+
					"requires the Node's be assigned a round to be in the "+
					"correct state; Assigned: %s, Expected: %s", oldActivity,
					newActivity, n.currentRound.GetRoundState(),
					transition.Node.GetValidRoundStateStrings(newActivity))
		}
	}

	//check that the Node doesn't have a round if it shouldn't
	if transition.Node.NeedsRound(newActivity) == transition.No && n.currentRound != nil {
		return false, UpdateNotification{},
			errors.Errorf("Node update from %s to %s failed, "+
				"requires the Node not be assigned a round", oldActivity,
				newActivity)
	}

	// change the Node's activity
	n.activity = newActivity
	// Timestamp of the last time this Node produced an update
	n.lastUpdate = time.Now()

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

// Gets the status of connectivity to the node, atomically
func (n *State) GetConnectivity() uint32 {
	// Done to avoid a race condition in the case of a double poll
	verify := atomic.CompareAndSwapUint32(n.connectivity, PortUnknown, PortVerifying)
	if verify {
		return PortUnknown
	} else {
		return atomic.LoadUint32(n.connectivity)
	}
}

// Gets the status of of the connectivity, but do not move from unknown
// to verifying
func (n *State) GetRawConnectivity() uint32 {
	return atomic.LoadUint32(n.connectivity)
}

// Sets the connectivity of node to c, atomically
func (n *State) SetConnectivity(c uint32) {
	atomic.StoreUint32(n.connectivity, c)
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

// gets the timestamp of the last time the Node updates
func (n *State) GetLastUpdate() time.Time {
	n.mux.RLock()
	defer n.mux.RUnlock()
	return n.lastUpdate
}

func (n *State) GetLastActive() time.Time {
	n.mux.Lock()
	defer n.mux.Unlock()
	return n.lastActive
}

func (n *State) SetLastActive() {
	n.mux.Lock()
	defer n.mux.Unlock()
	n.lastActive = time.Now()
}

func (n *State) SetLastActiveTesting(tm time.Time, x interface{}) {
	// Ensure that this function is only run in testing environments
	switch x.(type) {
	case *testing.T, *testing.M, *testing.B:
		break
	default:
		panic("SetLastActiveTesting() can only be used for testing.")
	}

	n.lastActive = tm
}

// Returns the polling lock
func (n *State) GetPollingLock() *sync.Mutex {
	return &n.pollingLock
}

// UpdateNodeAddresses updates the address if it is warranted.
func (n *State) UpdateNodeAddresses(node string) (bool, error) {
	n.mux.Lock()
	defer n.mux.Unlock()

	if n.nodeAddress == node {
		return false, nil
	}

	if time.Since(n.lastNodeUpdateTS) < ipUpdateTimeout {
		return false, errors.Errorf("cannot update node ip from %s to %s, can only "+
			"update every %s, last update was at %s", n.nodeAddress, node, ipUpdateTimeout, n.lastGatewayUpdateTS)
	}

	n.nodeAddress = node
	n.lastNodeUpdateTS = time.Now()

	return true, nil
}

// GetNodeAddresses return the current node address.
func (n *State) GetNodeAddresses() string {
	n.mux.RLock()
	defer n.mux.RUnlock()

	return n.nodeAddress
}

// UpdateGatewayAddresses updates the address if it is warranted
func (n *State) UpdateGatewayAddresses(gateway string) (bool, error) {
	n.mux.Lock()
	defer n.mux.Unlock()

	if gateway == "" || n.gatewayAddress == gateway {
		return false, nil
	}

	if time.Since(n.lastGatewayUpdateTS) < ipUpdateTimeout {
		return false, errors.Errorf("cannot update gateway ip from %s to %s, can only "+
			"update every %s, last update was at %s", n.gatewayAddress, gateway, ipUpdateTimeout, n.lastGatewayUpdateTS)
	}

	n.gatewayAddress = gateway
	n.lastGatewayUpdateTS = time.Now()

	return true, nil
}

// UpdateEd25519Key updates the ed25519 key used for no registration if warranted
func (n *State) UpdateEd25519Key(ed []byte) (bool, error) {
	n.mux.Lock()
	defer n.mux.Unlock()

	// Edge case in which we have an ED in state, but the poll gives us nil/empty
	if (ed == nil || len(ed) == 0) && n.ed25519 != nil {
		n.ed25519 = nil
		return true, nil
	}

	// If passed in ED matches the one in state, return false
	if ed == nil || (n.ed25519 != nil && bytes.Compare(n.ed25519.Bytes(), ed) == 0) {
		return false, nil
	}
	newEd, err := ecdh.ECDHNIKE.UnmarshalBinaryPublicKey(ed)
	if err != nil {
		return false, nil
	}
	n.ed25519 = newEd
	return true, nil
}

// GetOrdering return the ordering string for use in team formation.
func (n *State) GetOrdering() string {
	n.mux.RLock()
	defer n.mux.RUnlock()

	return n.ordering
}

// SetOrdering sets the State ordering string.
func (n *State) SetOrdering(ordering string) {
	n.mux.Lock()
	n.ordering = ordering
	n.mux.Unlock()
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
		return errors.Errorf("could not set the Node %s round when it is "+
			"already set, current round: %v, new round: %v", n.id,
			n.currentRound.GetRoundID(), r.GetRoundID())
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

func (n *State) GetGatewayAddress() string {
	return n.gatewayAddress
}
