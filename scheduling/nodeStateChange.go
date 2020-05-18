////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"time"
)

// HandleNodeUpdates handles the node state changes.
//  A node in waiting is added to the pool in preparation for precomputing.
//  A node in standby is added to a round in preparation for realtime.
//  A node in completed waits for all other nodes in the team to transition
//   before the round is updated.
func HandleNodeUpdates(update node.UpdateNotification, pool *waitingPool,
	state *storage.NetworkState, realtimeDelay time.Duration) error {
	// Check the round's error state
	n := state.GetNodeMap().GetNode(update.Node)

	// when a node poll is received, the nodes polling lock is taken.  If there
	// is no update, it is released in the endpoint, otherwise it is released
	// here which blocks all future polls until processing completes
	defer n.GetPollingLock().Unlock()

	hasRound, r := n.GetCurrentRound()
	roundErrored := hasRound == true && r.GetRoundState() == states.FAILED && update.ToActivity != current.ERROR
	if roundErrored {
		return nil
	}

	//ban the node if it is supposed to be banned
	if update.ToStatus == node.Banned {
		if hasRound {
			return killRound(state, r, n)
		} else {
			pool.Ban(n)
			return nil
		}
	}

	//get node and round information
	switch update.ToActivity {
	case current.NOT_STARTED:
		// Do nothing
	case current.WAITING:

		// Clear the round if node has one (it should unless it is
		// coming from NOT_STARTED
		if hasRound {
			n.ClearRound()
		}
		pool.Add(n)

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
		return killRound(state, r, n)
	}

	return nil
}

func killRound(state *storage.NetworkState, r *round.State, n *node.State) error {
	_ = r.Update(states.FAILED, time.Now())
	n.ClearRound()
	err := state.AddRoundUpdate(r.BuildRoundInfo())
	if err != nil {
		return errors.WithMessagef(err, "Could not issue "+
			"update to kill round %v", r.GetRoundID())
	}

	return nil
}
