////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
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

type protoRound struct {
	topology      *connect.Circuit
	ID            id.Round
	nodeStateList []*node.State
	batchSize     uint32
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
// state changes then creating a team from the nodes in the pool
func scheduler(params Params, state *storage.NetworkState) error {

	pool := newWaitingPool(state.GetNodeMap().Len())

	roundID := NewRoundID(0)
	errorChan := make(chan error, 1)
	newRoundChan := make(chan protoRound, 100)

	//begin the thread that starts rounds
	go func() {
		lastRound := time.Now()
		var err error
		for newRound := range newRoundChan {
			// To avoid back-to-back teaming, we make sure to sleep until the minimum delay
			if timeDiff := time.Now().Sub(lastRound); timeDiff < params.MinimumDelay {
				time.Sleep(timeDiff)
			}
			lastRound = time.Now()

			err = startRound(newRound, state, errorChan)
			if err != nil {
				break
			}
		}

		jww.ERROR.Printf("Round creation thread should never exit: %s", err)

	}()

	//start reciving updates from nodes
	for true {
		var update *storage.NodeUpdateNotification
		select {
		case err := <-errorChan:
			return err
		case update = <-state.GetNodeUpdateChannel():

		}

		//handle the node's state change
		err := HandleNodeStateChange(update, pool, state)
		if err != nil {
			return err
		}

		//create a new round if the pool is full
		if pool.Len() == int(params.TeamSize) {
			newRound, err := createRound(params, pool, roundID.Next(), state)
			if err != nil {
				return err
			}
			//send the round to the new round channel to be created
			newRoundChan <- newRound
		}

	}

	return errors.New("single scheduler should never exit")
}

func startRound(round protoRound, state *storage.NetworkState, errChan chan<- error) error {

	//create the round
	r, err := state.GetRoundMap().AddRound(round.ID, round.batchSize, round.topology)
	if err != nil {
		err = errors.WithMessagef(err, "Failed to create new round %v", round.ID)
		errChan <- err
		return err
	}

	//move the round to precomputing
	err = r.Update(states.PRECOMPUTING, time.Now())
	if err != nil {
		err = errors.WithMessagef(err, "Could not move new round into %s", states.PRECOMPUTING)
		errChan <- err
		return err
	}

	//tag all nodes to the round
	for _, n := range round.nodeStateList {
		err := n.SetRound(r)
		if err != nil {
			err = errors.WithMessagef(err, "could not add round %v to node %s", r.GetRoundID(), n.GetID())
			errChan <- err
			return err
		}
	}

	//issue the update
	err = state.AddRoundUpdate(r.BuildRoundInfo())
	if err != nil {
		err = errors.WithMessagef(err, "Could not issue "+
			"update to create round %v", r.GetRoundID())
		errChan <- err
		return err
	}

	return nil
}
