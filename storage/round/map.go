////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"github.com/golang-collections/collections/set"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"sync"
	"testing"
	"time"
)

// Tracks state of an individual Node in the network
type StateMap struct {
	mux sync.RWMutex

	rounds       map[id.Round]*State
	activeRounds *set.Set
}

//creates a state map object
func NewStateMap() *StateMap {
	return &StateMap{
		rounds:       make(map[id.Round]*State),
		activeRounds: set.New(),
	}
}

// Adds a new round state to the structure. Will not overwrite an existing one.
func (rsm *StateMap) AddRound(id id.Round, batchsize uint32, resourceQueueTimeout time.Duration,
	topology *connect.Circuit) (*State, error) {
	rsm.mux.Lock()
	defer rsm.mux.Unlock()

	if _, ok := rsm.rounds[id]; ok {
		return nil, errors.New("cannot add a round which already exists")
	}

	rsm.rounds[id] = newState(id, batchsize, resourceQueueTimeout, topology, time.Now())

	return rsm.rounds[id], nil
}

// Gets rounds from the state structure
func (rsm *StateMap) GetRound(id id.Round) *State {
	rsm.mux.RLock()
	defer rsm.mux.RUnlock()
	return rsm.rounds[id]
}

// Adds round id to active round tracker
func (rsm *StateMap) AddActiveRound(rid id.Round) {
	rsm.mux.RLock()
	defer rsm.mux.RUnlock()
	rsm.activeRounds.Insert(rid)
}

// Removes round from active round map
func (rsm *StateMap) RemoveActiveRound(rid id.Round) {
	rsm.mux.RLock()
	defer rsm.mux.RUnlock()

	rsm.activeRounds.Remove(rid)
}

// Gets the amount of active rounds in the set as well as the round id's
func (rsm *StateMap) GetActiveRounds() (int, []id.Round) {
	rsm.mux.RLock()
	defer rsm.mux.RUnlock()
	var rounds []id.Round
	rsm.activeRounds.Do(func(i interface{}) {
		rounds = append(rounds, i.(id.Round))
	})

	return rsm.activeRounds.Len(), rounds
}

//adds rounds for testing without checks
func (rsm *StateMap) AddRound_Testing(state *State, t *testing.T) {
	if t == nil {
		jww.FATAL.Panic("Only for testing")
	}

	rsm.rounds[state.GetRoundID()] = state

}
