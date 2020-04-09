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

	nodeStates map[id.Node]*State
}

func NewStateMap()*StateMap {
	return &StateMap{
		nodeStates: make(map[id.Node]*State),
	}
}

// Adds a new node state to the structure. Will not overwrite an existing one.
func (nsm *StateMap) AddNode(id *id.Node) error{
	nsm.mux.Lock()
	defer nsm.mux.Unlock()

	if _, ok :=nsm.nodeStates[*id]; ok{
		return errors.New("cannot add a node which already exists")
	}

	nsm.nodeStates[*id] =
		&State{
			activity:     current.NOT_STARTED,
			currentRound: nil,
			lastPoll:     time.Now(),
		}

	return nil
}

// Returns the State object for the given id if it exists
func (nsm *StateMap) GetNode(id *id.Node) *State {
	nsm.mux.RLock()
	defer nsm.mux.RUnlock()
	return nsm.nodeStates[*id]
}
