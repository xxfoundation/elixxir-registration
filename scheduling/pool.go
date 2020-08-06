////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"github.com/golang-collections/collections/set"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/registration/storage/node"
	"sync"
	"time"
)

// pool.go contains logic for the secure teaming algorithm's
//   waiting pool.

// Secure waiting pool struct. Contains 2 set objects.
// Pool holds nodes last seen as active. It may hold
//   offline nodes until properly cleaned, in which
//   case offline nodes are placed in the offline set
// Offline holds nodes found to be offline. Nodes need
//   to be manually set back to online with a function call
type waitingPool struct {
	pool    *set.Set
	offline *set.Set

	mux sync.RWMutex
}

// NewWaitingPool is a constructor for the waiting pool object
func NewWaitingPool() *waitingPool {
	return &waitingPool{
		pool:    set.New(),
		offline: set.New(),
	}
}

// Len returns the length of the online pool+
func (wp *waitingPool) Len() int {
	wp.mux.RLock()
	defer wp.mux.RUnlock()
	return wp.pool.Len()
}

// OfflineLen returns the length of the offline pool
func (wp *waitingPool) OfflineLen() int {
	wp.mux.RLock()
	defer wp.mux.RUnlock()
	return wp.offline.Len()
}

// Add inserts a node into the online pool
func (wp *waitingPool) Add(n *node.State) {
	wp.mux.Lock()
	wp.pool.Insert(n)
	wp.mux.Unlock()
}

// Removes the node from the pool banning it
func (wp *waitingPool) Ban(n *node.State) {
	wp.mux.Lock()
	wp.pool.Remove(n)
	wp.offline.Remove(n)
	wp.mux.Unlock()
}

// CleanOfflineNodes places all nodes with a lastPoll more than timeout
// into the offline pool
func (wp *waitingPool) CleanOfflineNodes(timeout time.Duration) {
	wp.mux.Lock()
	defer wp.mux.Unlock()
	now := time.Now()

	// Collect nodes whose lastPoll is longer than
	// timeout's duration
	var toRemove []*node.State
	wp.pool.Do(func(face interface{}) {
		ns := face.(*node.State)
		lastPoll := ns.GetLastPoll()
		if now.After(lastPoll) {
			delta := now.Sub(ns.GetLastPoll())
			if delta > timeout {
				toRemove = append(toRemove, ns)
			}
		}

	})

	for _, ns := range toRemove {
		jww.INFO.Printf("Node %v is offline. Removing from waiting pool", ns.GetID())
		ns.ClearRound()
		wp.pool.Remove(ns)
		wp.offline.Insert(ns)
		ns.SetInactive()
	}
}

// SetNodeToOnline removes a node from the offline pool and
//  inserts it into the online pool
func (wp *waitingPool) SetNodeToOnline(ns *node.State) {
	jww.TRACE.Printf("Node %v is online. Returning to waiting pool", ns.GetID())
	wp.mux.Lock()
	defer wp.mux.Unlock()

	wp.offline.Remove(ns)
	wp.pool.Insert(ns)
}

// PickNRandAtThreshold collects n nodes at random from the pool and returns
//   those nodes.
// If there are not enough nodes, either from the threshold or
//   the requested nodes, this function errors
func (wp *waitingPool) PickNRandAtThreshold(thresh, n int, disabledNodesSet *set.Set) ([]*node.State, error) {
	wp.mux.Lock()
	defer wp.mux.Unlock()

	newPool := set.New()

	// Filter disabled nodes from the list
	if disabledNodesSet != nil {
		newPool = wp.pool.Difference(disabledNodesSet)
	} else {
		newPool = wp.pool
	}

	// Check that the pool meets the threshold requirement
	if newPool.Len() < thresh {
		return nil, errors.Errorf("Number of stored nodes (%v) does not reach threshold", newPool.Len())
	}

	// Check that the pool has enough nodes to satisfy n
	if newPool.Len() < n {
		return nil, errors.Errorf("Number of stored nodes (%v) not enough"+
			" to pick %v nodes", newPool.Len(), n)
	}

	// Create an incrementing list of numbers up to pool's length
	numList := make([]uint32, newPool.Len())
	for i := 0; i < newPool.Len(); i++ {
		numList[i] = uint32(i)
	}

	// Shuffle these numbers
	shuffle.Shuffle32(&numList)

	var nodeList []*node.State
	iterator := 0

	// Collect nodes from pool at random
	newPool.Do(func(face interface{}) {
		if numList[iterator] < uint32(n) {
			nodeList = append(nodeList, face.(*node.State))
		}
		iterator++
	})

	// Remove collected nodes from pool
	for _, ns := range nodeList {
		wp.pool.Remove(ns)
	}

	// Return collected ndoes
	return nodeList, nil
}
