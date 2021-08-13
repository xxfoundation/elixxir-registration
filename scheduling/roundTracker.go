////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// This is used to track which rounds are currently running for debugging.

package scheduling

import (
	"git.xx.network/xx_network/primitives/id"
	"sync"
)

// RoundTracker tracks rounds that are active, meaning between precomputing and
// completed.
type RoundTracker struct {
	mux          sync.Mutex
	activeRounds map[id.Round]struct{}
}

// NewRoundTracker creates tracker object.
func NewRoundTracker() *RoundTracker {
	return &RoundTracker{
		activeRounds: make(map[id.Round]struct{}),
	}
}

// AddActiveRound adds round ID to active round tracker.
func (rt *RoundTracker) AddActiveRound(rid id.Round) {
	rt.mux.Lock()

	rt.activeRounds[rid] = struct{}{}

	rt.mux.Unlock()
}

// Len gives the number of members for the round tracker.
func (rt *RoundTracker) Len() int {
	rt.mux.Lock()
	defer rt.mux.Unlock()

	return len(rt.activeRounds)
}

// RemoveActiveRound removes round from active round map.
func (rt *RoundTracker) RemoveActiveRound(rid id.Round) {
	rt.mux.Lock()

	if _, exists := rt.activeRounds[rid]; exists {
		delete(rt.activeRounds, rid)
	}

	rt.mux.Unlock()
}

// GetActiveRounds gets the amount of active rounds in the set as well as the
// round IDs.
func (rt *RoundTracker) GetActiveRounds() []id.Round {
	var rounds []id.Round

	rt.mux.Lock()

	for rid := range rt.activeRounds {
		rounds = append(rounds, rid)
	}

	rt.mux.Unlock()

	return rounds
}
