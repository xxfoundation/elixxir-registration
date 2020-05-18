////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"time"
)

// startRound is a function which takes the info from createSimpleRound and updates the
//  node and network states in order to begin the round
func startRound(round protoRound, state *storage.NetworkState, errChan chan<- error) error {

	// Add the round to the manager
	r, err := state.GetRoundMap().AddRound(round.ID, round.batchSize, round.topology)
	if err != nil {
		err = errors.WithMessagef(err, "Failed to create new round %v", round.ID)
		errChan <- err
		return err
	}

	// Move the round to precomputing
	err = r.Update(states.PRECOMPUTING, time.Now())
	if err != nil {
		err = errors.WithMessagef(err, "Could not move new round into %s", states.PRECOMPUTING)
		errChan <- err
		return err
	}

	// Issue the update to the network state
	err = state.AddRoundUpdate(r.BuildRoundInfo())
	if err != nil {
		err = errors.WithMessagef(err, "Could not issue "+
			"update to create round %v", r.GetRoundID())
		errChan <- err
		return err
	}

	// Tag all nodes to the round
	for _, n := range round.nodeStateList {
		err := n.SetRound(r)
		if err != nil {
			err = errors.WithMessagef(err, "could not add round %v to node %s", r.GetRoundID(), n.GetID())
			errChan <- err
			return err
		}
	}

	return nil
}
