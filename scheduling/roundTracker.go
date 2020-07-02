////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

//this is used to track which rounds are currently running for debugging

package scheduling

import (
	"gitlab.com/elixxir/primitives/id"
	"sync"
)

// Tracks rounds which are active, meaning between precomputing and completed
type RoundTracker struct {
	mux          sync.Mutex
	activeRounds map[id.Round]struct{}
}

// Creates tracker object
func NewRoundTracker() *RoundTracker {
	return &RoundTracker{
		activeRounds: make(map[id.Round]struct{}),
	}
}

// Adds round id to active round tracker
func (rt *RoundTracker) AddActiveRound(rid id.Round) {
	rt.mux.Lock()

	rt.activeRounds[rid] = struct{}{}

	rt.mux.Unlock()
}

// gives the number of members for the round tracker
func (rt *RoundTracker) Len() int {
	rt.mux.Lock()
	defer rt.mux.Lock()

	return len(rt.activeRounds)
}

// Removes round from active round map
func (rt *RoundTracker) RemoveActiveRound(rid id.Round) {
	rt.mux.Lock()

	if _, exists := rt.activeRounds[rid]; exists {
		delete(rt.activeRounds, rid)
	}

	rt.mux.Unlock()
}

// Gets the amount of active rounds in the set as well as the round id's
func (rt *RoundTracker) GetActiveRounds() []id.Round {
	var rounds []id.Round

	rt.mux.Lock()

	for rid := range rt.activeRounds {
		rounds = append(rounds, rid)
	}

	rt.mux.Unlock()

	return rounds
}

/*
// tracks how many times the scheduler runs
type SchedulingTracker *uint32

func (sc SchedulingTracker)Incrememnt(){
	atomic.AddUint32()
}*/
