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

	// pool which tracks nodes which are not in a team
	pool := NewWaitingPool()

	//tracks and incrememnts the round id
	roundID := NewRoundID(0)

	//channel to send new rounds over to be created
	newRoundChan := make(chan protoRound, newRoundChanLen)

	//channel which the round creation thread returns errors on
	errorChan := make(chan error, 1)

	//calculate the realtime delay from params
	rtDelay := time.Duration(params.RealtimeDelay) * time.Millisecond

	//select the correct round creator
	var createRound roundCreator

	if params.Secure {
		jww.INFO.Printf("Using Secure Teaming Algorithm")
		createRound = createSecureRound
	} else {
		jww.INFO.Printf("Using Simple Teaming Algorithm")

		createRound = createSimpleRound
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

	//start receiving updates from nodes
	for true {
		var update node.UpdateNotification
		select {
		// If receive an error over a channel, return an error
		case err := <-errorChan:
			return err
		// when we get a node update, move base the select statement
		case update = <-state.GetNodeUpdateChannel():
		}

		//handle the node's state change
		err := HandleNodeUpdates(update, pool, state, rtDelay)
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
