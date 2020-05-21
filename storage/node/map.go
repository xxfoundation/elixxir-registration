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

type Status uint8

const (
	Unregistered = Status(iota) // Default state, equivalent to NULL
	Active                      // Operational, active node which will be considered for team
	Inactive                    // Inactive for a certain amount of time, not considered for teams
	Banned                      // Stop any teams and ban from teams until manually overridden
)

func (s Status) String() string {
	switch s {
	case Unregistered:
		return "Unregistered"
	case Active:
		return "Active"
	case Inactive:
		return "Inactive"
	case Banned:
		return "Banned"
	default:
		return "Unknown"
	}
}

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

// Adds a new node state to the structure. Will not overwrite an existing one.
func (nsm *StateMap) AddNode(id *id.ID, ordering, nAddr, gwAddr string) error {
	nsm.mux.Lock()
	defer nsm.mux.Unlock()

	if _, ok := nsm.nodeStates[*id]; ok {
		return errors.New("cannot add a node which already exists")
	}

	nsm.nodeStates[*id] =
		&State{
			activity:     current.NOT_STARTED,
			currentRound: nil,
			lastPoll:     time.Now(),
			ordering:     ordering,
			id:           id,
			nodeAddress: nAddr,
			gatewayAddress: gwAddr,
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
