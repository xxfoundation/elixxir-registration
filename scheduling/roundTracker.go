package scheduling

import (
	"github.com/golang-collections/collections/set"
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

type RoundTracker struct {
	mux          sync.Mutex
	activeRounds *set.Set
}

func NewRoundTracker() *RoundTracker {
	return &RoundTracker{
		mux:          sync.Mutex{},
		activeRounds: set.New(),
	}
}

// Adds round id to active round tracker
func (rsm *RoundTracker) AddActiveRound(rid id.Round) {
	rsm.mux.Lock()
	defer rsm.mux.Unlock()
	rsm.activeRounds.Insert(rid)
}

// Removes round from active round map
func (rsm *RoundTracker) RemoveActiveRound(rid id.Round) {
	rsm.mux.Lock()
	defer rsm.mux.Unlock()

	rsm.activeRounds.Remove(rid)
}

// Gets the amount of active rounds in the set as well as the round id's
func (rsm *RoundTracker) GetActiveRounds() []id.Round {
	rsm.mux.Lock()
	defer rsm.mux.Unlock()
	var rounds []id.Round
	rsm.activeRounds.Do(func(i interface{}) {
		rounds = append(rounds, i.(id.Round))
	})

	return rounds
}
