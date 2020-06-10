////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"encoding/json"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"time"
)

// scheduler.go contains the business logic for scheduling a round

//size of round creation channel, just sufficiently large enough to not be jammed
const newRoundChanLen = 100

type roundCreator func(params Params, pool *waitingPool, roundID id.Round,
	state *storage.NetworkState) (protoRound, error)

// Scheduler constructs the teaming parameters and sets up the scheduling
func Scheduler(serialParam []byte, state *storage.NetworkState, killchan chan chan struct{}) error {
	var params Params
	err := json.Unmarshal(serialParam, &params)
	if err != nil {
		return errors.WithMessage(err, "Could not extract parameters")
	}

	return scheduler(params, state, killchan)
}

// scheduler is a utility function which builds a round by handling a node's
// state changes then creating a team from the nodes in the pool
func scheduler(params Params, state *storage.NetworkState, killchan chan chan struct{}) error {

	// pool which tracks nodes which are not in a team
	pool := NewWaitingPool()

	//channel to send new rounds over to be created
	newRoundChan := make(chan protoRound, newRoundChanLen)

	//channel which the round creation thread returns errors on
	errorChan := make(chan error, 1)

	//calculate the realtime delay from params
	rtDelay := params.RealtimeDelay * time.Millisecond

	//select the correct round creator
	var createRound roundCreator
	var teamFormationThreshold uint32

	if params.Secure {
		jww.INFO.Printf("Using Secure Teaming Algorithm")
		createRound = createSecureRound
		teamFormationThreshold = params.Threshold
	} else {
		jww.INFO.Printf("Using Simple Teaming Algorithm")
		createRound = createSimpleRound
		teamFormationThreshold = params.TeamSize
	}

	//begin the thread that starts rounds
	go func() {
		lastRound := time.Now()
		var err error
		for newRound := range newRoundChan {
			// To avoid back-to-back teaming, we make sure to sleep until the minimum delay
			if timeDiff := time.Now().Sub(lastRound); timeDiff < params.MinimumDelay*time.Millisecond {
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

	var killed chan struct{}

	numRounds := 0

	// Uncomment when need to debug status of rounds
	//go trackRounds(params, state, pool)

	//start receiving updates from nodes
	for true {
		var update node.UpdateNotification
		select {
		// receive a signal to kill the scheduler
		case killed = <-killchan:
		// If receive an error over a channel, return an error
		case err := <-errorChan:
			return err
		// when we get a node update, move base the select statement
		case update = <-state.GetNodeUpdateChannel():
		}

		//handle the node's state change
		endRound, err := HandleNodeUpdates(update, pool, state, rtDelay)
		if err != nil {
			return err
		}

		//remove offline nodes from pool to more accurately determine if pool is eligible for round creation
		pool.CleanOfflineNodes(params.NodeCleanUpInterval * time.Second)

		//if a round has finished, decrement num rounds
		if endRound {
			numRounds--
		}

		//create a new round if the pool is full
		if pool.Len() >= int(teamFormationThreshold) && killed == nil {
			// Increment round ID
			currentID, err := state.IncrementRoundID()
			if err != nil {
				return err
			}

			newRound, err := createRound(params, pool, currentID, state)
			if err != nil {
				return err
			}

			//send the round to the new round channel to be created
			newRoundChan <- newRound
			numRounds++
		}

		// if the scheduler is to be killed and no rounds are in progress,
		// kill the scheduler
		if killed != nil && numRounds == 0 {
			close(newRoundChan)
			killed <- struct{}{}
			return nil
		}

	}

	return errors.New("single scheduler should never exit")
}

// Tracks rounds, periodically outputs how many teams are in various rounds
func trackRounds(params Params, state *storage.NetworkState, pool *waitingPool) {
	// Period of polling the state map for logs
	schedulingTicker := time.NewTicker(15 * time.Second)

	realtimeNodes := make([]*node.State, 0)
	precompNodes := make([]*node.State, 0)
	waitingNodes := make([]*node.State, 0)

	for {
		select {
		case <-schedulingTicker.C:
			// Parse through the node map to collect nodes into round state arrays
			nodeStates := state.GetNodeMap().GetNodeStates()
			for _, nodeState := range nodeStates {
				switch nodeState.GetActivity() {
				case current.WAITING:
					waitingNodes = append(waitingNodes, nodeState)
				case current.REALTIME:
					realtimeNodes = append(realtimeNodes, nodeState)
				case current.PRECOMPUTING:
					precompNodes = append(precompNodes, nodeState)
				}

			}

		}

		// Output data into logs
		jww.TRACE.Printf("Nodes in realtime: %v", len(realtimeNodes)/int(params.TeamSize))
		jww.TRACE.Printf("Nodes in precomp: %v", len(precompNodes)/int(params.TeamSize))
		jww.TRACE.Printf("Nodes in waiting: %v", len(waitingNodes)/int(params.TeamSize))
		jww.TRACE.Printf("Nodes in pool: %v", pool.Len())
		jww.TRACE.Printf("Nodes in offline pool: %v", pool.OfflineLen())

		// Reset the data for next periodic poll
		realtimeNodes = make([]*node.State, 0)
		precompNodes = make([]*node.State, 0)
		waitingNodes = make([]*node.State, 0)

	}

}
