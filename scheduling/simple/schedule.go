////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
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
	errorChan := make(chan error)

	go func() {
		startRound(state, params, topology, newRound, updateID, nodeStateList, nodes, errorChan)
	}()

	for true {
		var update *storage.NodeUpdateNotification
		select {
		case err := <-errorChan:
			return err
		case update = <-state.GetNodeUpdateChannel():

		}

		//handle the node's state change
		err := HandleNodeStateChange(update, pool, updateID, state)
		if err != nil {
			return err
		}

		var topology *connect.Circuit
		var nodeStateList []*node.State
		var newRound id.Round
		var nodes []*id.Node

		//create a new round if the pool is full
		if pool.Len() == int(params.TeamSize) {
			topology, newRound, nodeStateList, nodes, err = createRound(params, pool, roundID, state)
			if err != nil {
				return err
			}
		}

	}

	return errors.New("single scheduler should never exit")
}

func startRound(state *storage.NetworkState, params Params,
	topology *connect.Circuit, roundID id.Round, updateID *UpdateID,
	nodeStateList []*node.State, nodes []*id.Node, errChan chan error) {

	// To avoid back-to-back teaming, we make sure to sleep until the minimum delay
	if timeDiff := time.Now().Sub(params.LastRound); timeDiff < params.MinimumDelay {
		time.Sleep(timeDiff)
	}

	//create the round
	r, err := state.GetRoundMap().AddRound(roundID, params.BatchSize, topology)
	if err != nil {
		errChan <- errors.WithMessagef(err, "Failed to create new round %v", roundID)
	}

	//move the round to precomputing
	err = r.Update(states.PRECOMPUTING, time.Now())
	if err != nil {
		errChan <- errors.WithMessagef(err, "Could not move new round into %s", states.PRECOMPUTING)
	}

	//tag all nodes to the round
	for i, n := range nodeStateList {
		err := n.SetRound(r)
		if err != nil {
			errChan <- errors.WithMessagef(err, "could not add round %v to node %s", r.GetRoundID(), nodes[i])
		}
	}

	//issue the update
	err = state.AddRoundUpdate(updateID.Next(), r.BuildRoundInfo())
	if err != nil {
		errChan <- errors.WithMessagef(err, "Could not issue "+
			"update to create round %v", r.GetRoundID())
	}
}
