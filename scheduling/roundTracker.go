package scheduling

import (
	"github.com/golang-collections/collections/set"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

// Tracks rounds which are active, meaning between precomputing and completed
type RoundTracker struct {
	mux          sync.Mutex
	activeRounds *set.Set
}

// Creates tracker object
func NewRoundTracker() *RoundTracker {
	return &RoundTracker{
		activeRounds: set.New(),
	}
}

// Adds round id to active round tracker
func (rt *RoundTracker) AddActiveRound(rid id.Round) {
	rt.mux.Lock()
	defer rt.mux.Unlock()
	rt.activeRounds.Insert(rid)
}

// Removes round from active round map
func (rt *RoundTracker) RemoveActiveRound(rid id.Round) {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	rt.activeRounds.Remove(rid)
}

// Gets the amount of active rounds in the set as well as the round id's
func (rt *RoundTracker) GetActiveRounds() []id.Round {
	rt.mux.Lock()
	defer rt.mux.Unlock()
	var rounds []id.Round
	rt.activeRounds.Do(func(i interface{}) {
		rounds = append(rounds, i.(id.Round))
	})

	return rounds
}
