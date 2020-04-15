////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/registration/storage"
	"time"
)

// scheduler.go contains the business logic for scheduling a round

type Params struct {
	TeamSize       uint32
	BatchSize      uint32
	RandomOrdering bool
	MinimumDelay   time.Duration
	LastRound      time.Time
}

// Scheduler constructs the teaming parameters and sets up the scheduling
func Scheduler(serialParam []byte, state *storage.NetworkState) error {
	var params Params
	err := json.Unmarshal(serialParam, &params)
	if err != nil {
		return errors.WithMessage(err, "Could not extract parameters")
	}

	return scheduler(params, state)
}

// scheduler is a utility function which builds a round by handling a node's
//  state changes then creating a team from the nodes in the pool
func scheduler(params Params, state *storage.NetworkState) error {

	pool := newWaitingPool(state.GetNodeMap().Len())

	roundID := NewRoundID(0)
	updateID := NewUpdateID(0)

	for update := range state.GetNodeUpdateChannel() {

		// To avoid back-to-back teaming, we make sure to sleep until the minimum delay
		if timeDiff := time.Now().Sub(params.LastRound); timeDiff < params.MinimumDelay {
			time.Sleep(timeDiff)
		}

		//handle the node's state change
		err := HandleNodeStateChange(update, pool, updateID, state)
		if err != nil {
			return err
		}

		//create a new round if the pool is full
		if pool.Len() == int(params.TeamSize) {
			err = createRound(params, pool, roundID, updateID, state)
			if err != nil {
				return err
			}
		}

	}

	return errors.New("single scheduler should never exit")
}
