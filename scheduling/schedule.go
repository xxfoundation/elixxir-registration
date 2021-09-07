////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
	"sync/atomic"
	"time"
)

// scheduler.go contains the business logic for scheduling a round

//size of round creation channel, just sufficiently large enough to not be jammed
const newRoundChanLen = 1000

type roundCreator func(params Params, pool *waitingPool, roundID id.Round,
	state *storage.NetworkState) (protoRound, error)

// Scheduler constructs the teaming parameters and sets up the scheduling
func Scheduler(serialParam []byte, state *storage.NetworkState, killchan chan chan struct{}) error {
	var params *Params
	err := json.Unmarshal(serialParam, params)
	if err != nil {
		return errors.WithMessage(err, "Could not extract parameters")
	}

	// If resource queue timeout isn't set, set it to a default of 3 minutes
	if params.ResourceQueueTimeout == 0 {
		params.ResourceQueueTimeout = 180000 // 180000 ms = 3 minutes
	}
	// If roundTimeout hasn't set, set to a default of one minute
	if params.PrecomputationTimeout == 0 {
		params.PrecomputationTimeout = 60
	}

	if params.RealtimeTimeout == 0 {
		params.RealtimeTimeout = 15
	}

	// TODO: Set up frequency
	go updateParams(params, 1*time.Minute)

	return scheduler(params, state, killchan)
}

// Runs an infinite loop that checks for updates to scheduling parameters
func updateParams(params *Params, updateFreq time.Duration) {
	for {
		teamSize, err := storage.PermissioningDb.GetStateInt(storage.TeamSize)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		batchSize, err := storage.PermissioningDb.GetStateInt(storage.BatchSize)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		precompTimeout, err := storage.PermissioningDb.GetStateInt(storage.PrecompTimeout)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		realtimeTimeout, err := storage.PermissioningDb.GetStateInt(storage.RealtimeTimeout)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		minDelay, err := storage.PermissioningDb.GetStateInt(storage.MinDelay)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}

		jww.INFO.Printf("Preparing to update scheduling params...")
		params.Lock()
		jww.INFO.Printf("Updating scheduling params...")
		params.TeamSize = uint32(teamSize)
		params.BatchSize = uint32(batchSize)
		params.PrecomputationTimeout = time.Duration(precompTimeout)
		params.RealtimeTimeout = time.Duration(realtimeTimeout)
		params.MinimumDelay = time.Duration(minDelay)
		params.Unlock()

		time.Sleep(updateFreq)
	}

}

// scheduler is a utility function which builds a round by handling a node's
// state changes then creating a team from the nodes in the pool
func scheduler(params *Params, state *storage.NetworkState, killchan chan chan struct{}) error {

	// Pool which tracks nodes which are not in a team
	pool := NewWaitingPool()

	// Channel to send new rounds over to be created
	newRoundChan := make(chan protoRound, newRoundChanLen)

	// Select the correct round creator
	var createRound roundCreator

	// Identify which teaming algorithm we will be using
	if params.Secure {
		jww.INFO.Printf("Using Secure Teaming Algorithm")
		createRound = createSecureRound
	} else {
		jww.INFO.Printf("Using Simple Teaming Algorithm")
		createRound = createSimpleRound
	}

	// Channel to communicate that a round has timed out
	roundTimeoutTracker := make(chan id.Round, 1000)

	roundTracker := NewRoundTracker()

	//begin the thread that starts rounds
	go func() {
		params.RLock()
		paramsCopy := *params
		params.RUnlock()

		lastRound := time.Now()
		var err error
		minRoundDelay := paramsCopy.MinimumDelay * time.Millisecond
		for newRound := range newRoundChan {

			// To avoid back-to-back teaming, we make sure to sleep until the minimum delay
			if timeDiff := time.Now().Sub(lastRound); timeDiff < minRoundDelay {
				time.Sleep(minRoundDelay - timeDiff)
			}
			lastRound = time.Now()

			ourRound, err := startRound(newRound, state, roundTracker)
			if err != nil {
				jww.FATAL.Panicf("Failed to start round %v: %+v", newRound.ID, err)
			}

			go waitForRoundTimeout(roundTimeoutTracker, state, ourRound,
				paramsCopy.PrecomputationTimeout*time.Second,
				"precomputation")
		}

		jww.FATAL.Panicf("Round creation thread should never exit: %v", err)

	}()

	var killed chan struct{}
	iterationsCount := uint32(0)

	// optional debug print which regularly prints the status of rounds and nodes
	// turned on by setting DebugTrackRounds to true in the scheduling config
	if params.DebugTrackRounds {
		go trackRounds(state, pool, roundTracker, &iterationsCount)
	}

	// Start receiving updates from nodes
	for true {
		params.RLock()
		paramsCopy := *params
		params.RUnlock()

		isRoundTimeout := false
		var update node.UpdateNotification
		var timedOutRoundID id.Round
		hasUpdate := false

		select {
		// Receive a signal to kill the scheduler
		case killed = <-killchan:
			// Also kill the unsticker
			jww.WARN.Printf("Scheduler has received a kill signal, exit process has begun")
		// When we get a node update, move past the select statement
		case update = <-state.GetNodeUpdateChannel():
			hasUpdate = true
		// Receive a signal indicating that a round has timed out
		case timedOutRoundID = <-roundTimeoutTracker:
			isRoundTimeout = true
		}

		atomic.AddUint32(&iterationsCount, 1)
		if isRoundTimeout {
			// Handle the timed out round
			err := timeoutRound(state, timedOutRoundID, roundTracker)
			if err != nil {
				return err
			}
		} else if hasUpdate {
			var err error

			// Handle the node's state change
			err = HandleNodeUpdates(update, pool, state,
				paramsCopy.RealtimeDelay*time.Millisecond, roundTracker, roundTimeoutTracker,
				paramsCopy.RealtimeTimeout*time.Second)
			if err != nil {
				return err
			}
		}

		for {
			//get the pool of disabled nodes and determine how many
			//nodes can be scheduled
			numNodesInPool := pool.Len()

			// Create a new round if the pool is full
			var teamFormationThreshold uint32
			if paramsCopy.Secure {
				teamFormationThreshold = paramsCopy.Threshold
			} else {
				teamFormationThreshold = paramsCopy.TeamSize
			}
			if numNodesInPool >= int(teamFormationThreshold) && killed == nil {

				// Increment round ID
				currentID, err := state.IncrementRoundID()

				if err != nil {
					return err
				}

				newRound, err := createRound(paramsCopy, pool, currentID, state)
				if err != nil {
					return err
				}

				// Send the round to the new round channel to be created
				newRoundChan <- newRound
			} else {
				break
			}
		}

		// If the scheduler is to be killed and no rounds are in progress,
		// kill the scheduler
		if killed != nil && roundTracker.Len() == 0 {
			// Stop round creation
			close(newRoundChan)
			jww.WARN.Printf("Scheduler is exiting due to kill signal")
			killed <- struct{}{}
			return nil
		}
	}

	return errors.New("single scheduler should never exit")
}

// Helper function which handles when we receive a timed out round
func timeoutRound(state *storage.NetworkState, timeoutRoundID id.Round,
	roundTracker *RoundTracker) error {
	// On a timeout, check if the round is completed. If not, kill it
	ourRound := state.GetRoundMap().GetRound(timeoutRoundID)
	roundState := ourRound.GetRoundState()

	// If the round is neither in completed or failed
	if roundState != states.COMPLETED && roundState != states.FAILED {

		timeoutType := "precomputation"

		if roundState > states.PRECOMPUTING {
			timeoutType = "realtime"
		}

		// Build the round error message
		timeoutError := &pb.RoundError{
			Id:     uint64(ourRound.GetRoundID()),
			NodeId: id.Permissioning.Marshal(),
			Error: fmt.Sprintf("Round %d killed due to a %s "+
				"round time out", ourRound.GetRoundID(), timeoutType),
		}

		// Sign the error message with our private key
		err := signature.SignRsa(timeoutError, state.GetPrivateKey())
		if err != nil {
			jww.FATAL.Panicf("Failed to sign error message for "+
				"%s timed out round %d: %+v", timeoutType,
				ourRound.GetRoundID(), err)
		}

		err = killRound(state, ourRound, timeoutError, roundTracker)
		if err != nil {
			return errors.WithMessagef(err, "Failed to kill round %d: %s",
				ourRound.GetRoundID(), err)
		}
	}
	return nil
}

// how long a node needs to not act to be considered offline or in-active for the
// print. arbitrarily chosen.
const timeToInactive = 3 * time.Minute
