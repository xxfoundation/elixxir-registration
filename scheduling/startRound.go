////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package scheduling

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/spf13/jwalterweatherman"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/round"
	"time"
)

// startRound is a function which takes the info from createSimpleRound and updates the
//  node and network states in order to begin the round
func startRound(round protoRound, state *storage.NetworkState, roundTracker *RoundTracker) (*round.State, error) {
	// Add the round to the manager
	r, err := state.GetRoundMap().AddRound(round.ID, round.BatchSize, state.GetAddressSpaceSize(), round.ResourceQueueTimeout,
		round.Topology)
	if err != nil {
		err = errors.WithMessagef(err, "Failed to create new round %v", round.ID)
		return nil, err
	}

	// Move the round to precomputing
	err = r.Update(states.PRECOMPUTING, time.Now())
	if err != nil {
		err = errors.WithMessagef(err, "Could not move new round into %s", states.PRECOMPUTING)
		return nil, err
	}

	// Tag all nodes to the round
	for i, n := range round.NodeStateList {
		jwalterweatherman.TRACE.Printf("Node %v is (%d)/(%d) of round",
			round.Topology.GetNodeAtIndex(i), i, len(round.NodeStateList))
		err = n.SetRound(r)
		if err != nil {
			return nil, errors.WithMessagef(err, "could not add round %v to node %s", r.GetRoundID(), n.GetID())
		}
	}

	// Issue the update to the network state
	err = state.AddRoundUpdate(r.BuildRoundInfo())
	if err != nil {
		err = errors.WithMessagef(err, "Could not issue "+
			"update to create round %v", r.GetRoundID())
		return nil, err
	}

	// Add round to active set of rounds
	roundTracker.AddActiveRound(r.GetRoundID())

	//print the round to the log
	roundPrnt := fmt.Sprintf("Scheduling round %d with nodes: ", round.ID)
	for i := 0; i < round.Topology.Len(); i++ {
		roundPrnt += fmt.Sprintf("\n\t (%d/%d) %s", i+1, round.Topology.Len(), round.Topology.GetNodeAtIndex(i))
	}
	jww.DEBUG.Println(roundPrnt)

	return r, nil
}
