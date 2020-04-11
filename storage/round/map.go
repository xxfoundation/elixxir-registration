package round

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"sync"
	"time"
)

// Tracks state of an individual Node in the network
type StateMap struct {
	mux sync.RWMutex

	rounds map[id.Round]*State
}

func NewStateMap() *StateMap {
	return &StateMap{
		rounds: make(map[id.Round]*State),
	}
}

// Adds a new round state to the structure. Will not overwrite an existing one.
func (rsm *StateMap) AddRound(id id.Round, batchsize uint32,
	topology *connect.Circuit) (*State, error) {
	rsm.mux.Lock()
	defer rsm.mux.Unlock()

	if _, ok := rsm.rounds[id]; ok {
		return nil, errors.New("cannot add a round which already exists")
	}

	rsm.rounds[id] = newState(id, batchsize, topology, time.Now())

	return rsm.rounds[id], nil
}

// Gets rounds from the state structure
func (rsm *StateMap) GetRound(id id.Round) *State {
	rsm.mux.RLock()
	rsm.mux.RUnlock()
	return rsm.rounds[id]
}
