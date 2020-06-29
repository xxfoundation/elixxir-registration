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
func (nsm *StateMap) AddNode(id *id.ID, ordering, nAddr, gwAddr string, appID uint64) error {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()

	if _, ok := nsm.nodeStates[*id]; ok {
		return errors.New("cannot add a Node which already exists")
	}
	pfState := PortUnknown

	numPolls := uint64(0)
	nsm.nodeStates[*id] =
		&State{
			activity:       current.NOT_STARTED,
			currentRound:   nil,
			lastPoll:       time.Now(),
			ordering:       ordering,
			id:             id,
			nodeAddress:    nAddr,
			gatewayAddress: gwAddr,
			status:         Active,
			numPolls:       &numPolls,
			mux:            sync.RWMutex{},
			connectivity:   &pfState,
			applicationID:  appID,
		}

	return nil
}

// Adds a new Node state to the structure. Will not overwrite an existing one.
func (nsm *StateMap) AddBannedNode(id *id.ID, ordering, nAddr, gwAddr string) error {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()

	if _, ok := nsm.nodeStates[*id]; ok {
		return errors.New("cannot add a Node which already exists")
	}

	numPolls := uint64(0)
	nsm.nodeStates[*id] =
		&State{
			activity:       current.NOT_STARTED,
			currentRound:   nil,
			lastPoll:       time.Now(),
			ordering:       ordering,
			id:             id,
			nodeAddress:    nAddr,
			gatewayAddress: gwAddr,
			status:         Banned,
			numPolls:       &numPolls,
			mux:            sync.RWMutex{},
		}

	return nil
}

// Returns the State object for the given id if it exists
func (nsm *StateMap) GetNode(id *id.ID) *State {
	nsm.mux.RLock()
	defer nsm.mux.RUnlock()
	return nsm.nodeStates[*id]
}

// Returns a list of all node States in the nsm
func (nsm *StateMap) GetNodeStates() []*State {
	nodeStates := make([]*State, len(nsm.nodeStates))
	idx := 0

	nsm.mux.RLock()
	defer nsm.mux.RUnlock()
	for _, nodeState := range nsm.nodeStates {
		nodeStates[idx] = nodeState
		idx++
	}
	return nodeStates
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
