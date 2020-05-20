////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"sync"
	"time"
)

// Tracks state of an individual Node in the network
type StateMap struct {
	mux sync.RWMutex

	nodeStates map[id.ID]*State
}

func NewStateMap() *StateMap {
	return &StateMap{
		nodeStates: make(map[id.ID]*State),
	}
}

// Adds a new Node state to the structure. Will not overwrite an existing one.
func (nsm *StateMap) AddNode(id *id.ID, ordering string) error {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()

	if _, ok := nsm.nodeStates[*id]; ok {
		return errors.New("cannot add a Node which already exists")
	}

	nsm.nodeStates[*id] =
		&State{
			activity:     current.NOT_STARTED,
			currentRound: nil,
			lastPoll:     time.Now(),
			ordering:     ordering,
			id:           id,
			status:       Active,
			mux:          sync.RWMutex{},
		}

	return nil
}

// Returns the State object for the given id if it exists
func (nsm *StateMap) GetNode(id *id.ID) *State {
	nsm.mux.RLock()
	defer nsm.mux.RUnlock()
	return nsm.nodeStates[*id]
}

// Commits metrics for all current Nodes to storage
// Takes the time period between writing metrics as an argument
func (nsm *StateMap) WriteNodeMetrics(interval time.Duration) error {
	for nodeId, nodeState := range nsm.nodeStates {

		// Build the NodeMetric
		currentTime := time.Now()
		metric := storage.NodeMetric{
			NodeId: nodeId.String(),
			// Subtract duration from current time to get start time
			StartTime: currentTime.Add(-interval),
			EndTime:   currentTime,
			NumPings:  nodeState.numPolls,
		}

		// Store the NodeMetric
		err := storage.PermissioningDb.InsertNodeMetric(metric)
		if err != nil {
			return err
		}

		// Reset Node polling data
		nodeState.ResetNumPolls()
	}

	return nil
}

// Returns the number of elements in the NodeMapo
func (nsm *StateMap) Len() int {
	nsm.mux.RLock()
	defer nsm.mux.RUnlock()
	num := 0
	for range nsm.nodeStates {
		num++
	}
	return num
}
