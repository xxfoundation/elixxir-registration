////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"time"
)

// HandleNodeStateChange handles the node state changes.
//  A node in waiting is added to the pool in preparation for precomputing.
//  A node in standby is added to a round in preparation for realtime.
//  A node in completed waits for all other nodes in the team to transition
//   before the round is updated.
func HandleNodeStateChange(update *storage.NodeUpdateNotification, pool *waitingPoll,
	state *storage.NetworkState, realtimeDelay time.Duration) error {
	// Check the round's error state
	n := state.GetNodeMap().GetNode(update.Node)

	// when a node poll is received, the nodes polling lock is taken.  If there
	// is no update, it is released in the endpoint, otherwise it is released
	// here which blocks all future polls until processing completes
	defer n.GetPollingLock().Unlock()

	hasRound, r := n.GetCurrentRound()
	roundErrored := hasRound == true && r.GetRoundState() == states.FAILED && update.To != current.ERROR
	if roundErrored {
		return nil
	}

	//get node and round information
	switch update.To {
	case current.NOT_STARTED:
		// Do nothing
	case current.WAITING:

		// Clear the round if node has one (it should unless it is
		// coming from NOT_STARTED
		if hasRound {
			n.ClearRound()
		}
		err := pool.Add(update.Node)
		if err != nil {
			return errors.WithMessage(err, "Waiting pool should never fill")
		}
	case current.PRECOMPUTING:
		// Do nothing
	case current.STANDBY:

		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be in %s state", update.Node, states.PRECOMPUTING)
		}

		stateComplete := r.NodeIsReadyForTransition()
		if stateComplete {
			err := r.Update(states.REALTIME, time.Now().Add(realtimeDelay))
			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.PRECOMPUTING, states.REALTIME)
			}
			err = state.AddRoundUpdate(r.BuildRoundInfo())
			if err != nil {
				return errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.PRECOMPUTING, states.REALTIME)
			}
		}
	case current.REALTIME:
		// Do nothing
	case current.COMPLETED:

		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be in %s state", update.Node, states.COMPLETED)
		}

		n.ClearRound()

		stateComplete := r.NodeIsReadyForTransition()
		if stateComplete {
			err := r.Update(states.COMPLETED, time.Now())
			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}
			err = state.AddRoundUpdate(r.BuildRoundInfo())
			if err != nil {
				return errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}
		}
	case current.ERROR:
		_ = r.Update(states.FAILED, time.Now())
		n.ClearRound()
		err := state.AddRoundUpdate(r.BuildRoundInfo())
		if err != nil {
			return errors.WithMessagef(err, "Could not issue "+
				"update for round %v transitioning from %s to %s",
				r.GetRoundID(), update.From, update.To)
		}
	}

	return nil
}
