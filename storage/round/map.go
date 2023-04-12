////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"testing"
	"time"
)

// StateMap tracks state of an individual Node in the network
type StateMap struct {
	mux sync.RWMutex

	rounds map[id.Round]*State
}

// NewStateMap creates a state map object
func NewStateMap() *StateMap {
	return &StateMap{
		rounds: make(map[id.Round]*State),
	}
}

// AddRound adds a new round state to the structure.
// Will not overwrite an existing one.
func (rsm *StateMap) AddRound(id id.Round, batchSize, addressSpaceSize uint32, resourceQueueTimeout time.Duration,
	topology *connect.Circuit) (*State, error) {
	rsm.mux.Lock()
	defer rsm.mux.Unlock()

	if _, ok := rsm.rounds[id]; ok {
		return nil, errors.New("cannot add a round which already exists")
	}

	rsm.rounds[id] = newState(id, batchSize, addressSpaceSize, resourceQueueTimeout, topology, time.Now())
	jww.TRACE.Printf("Added round %s to StateMap[%d]", id, len(rsm.rounds))
	return rsm.rounds[id], nil
}

// GetRound obtains round State from the state structure
func (rsm *StateMap) GetRound(id id.Round) (*State, bool) {
	rsm.mux.RLock()
	defer rsm.mux.RUnlock()
	s, exists := rsm.rounds[id]
	return s, exists
}

// add a schedule to delete timestamp

// DeleteRound cleans out rounds from round map.
// ONLY to be used upon round completion, failure, or timeout.
func (rsm *StateMap) DeleteRound(id id.Round) {
	// Delete the round from the map
	rsm.mux.Lock()
	delete(rsm.rounds, id)
	rsm.mux.Unlock()
	return
}

// AddRound_Testing adds rounds for testing without checks
func (rsm *StateMap) AddRound_Testing(state *State, t *testing.T) {
	if t == nil {
		jww.FATAL.Panic("Only for testing")
	}

	rsm.rounds[state.GetRoundID()] = state

}
