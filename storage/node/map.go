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
func (nsm *StateMap) AddNode(id *id.ID, ordering, nAddr, gwAddr string) error {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()

	if _, ok := nsm.nodeStates[*id]; ok {
		return errors.New("cannot add a Node which already exists")
	}

	nsm.nodeStates[*id] =
		&State{
			activity:       current.NOT_STARTED,
			currentRound:   nil,
			lastPoll:       time.Now(),
			ordering:       ordering,
			id:             id,
			nodeAddress:    nAddr,
			gatewayAddress: gwAddr,
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
