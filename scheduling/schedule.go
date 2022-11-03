////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package scheduling

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/fastRNG"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/csprng"
	"gitlab.com/xx_network/primitives/id"
	"io"
	"runtime"
	"strconv"
	"sync/atomic"
	"time"
)

// Scheduler.go contains the business logic for scheduling a round

const (
	//size of round creation channel, just sufficiently large enough to not be jammed
	newRoundChanLen = 1000

	// how long a node needs to not act to be considered offline or in-active for the
	// print. arbitrarily chosen.
	timeToInactive = 3 * time.Minute
)

type roundCreator func(params Params, pool *waitingPool, threshold int, roundID id.Round,
	state *storage.NetworkState, rng io.Reader) (protoRound, error)

func ParseParams(serialParam []byte) *SafeParams {
	// Parse params JSON
	params := &SafeParams{}
	err := json.Unmarshal(serialParam, params)
	if err != nil {
		jww.FATAL.Panicf("Scheduling Algorithm exited: Could not extract parameters")
	}

	// If resource queue timeout isn't set, set it to a default of 3 minutes
	if params.ResourceQueueTimeout == 0 {
		params.ResourceQueueTimeout = 180000
	}
	// If round times haven't been set, set to a default of one minute
	if params.PrecomputationTimeout == 0 {
		params.PrecomputationTimeout = 60000
	}
	if params.RealtimeTimeout == 0 {
		params.RealtimeTimeout = 15000
	}

	return params
}

// Runs an infinite loop that checks for updates to scheduling parameters
func UpdateParams(params *SafeParams, updateFreq time.Duration) {
	for {
		newParams := make(map[string]uint64, 0)
		teamSize, err := storage.PermissioningDb.GetStateInt(storage.TeamSize)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		newParams[storage.TeamSize] = teamSize
		batchSize, err := storage.PermissioningDb.GetStateInt(storage.BatchSize)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		newParams[storage.BatchSize] = batchSize
		precompTimeout, err := storage.PermissioningDb.GetStateInt(storage.PrecompTimeout)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		newParams[storage.PrecompTimeout] = precompTimeout
		realtimeTimeout, err := storage.PermissioningDb.GetStateInt(storage.RealtimeTimeout)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		newParams[storage.RealtimeTimeout] = realtimeTimeout
		minDelay, err := storage.PermissioningDb.GetStateInt(storage.MinDelay)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		newParams[storage.MinDelay] = minDelay
		realtimeDelay, err := storage.PermissioningDb.GetStateInt(storage.AdvertisementTimeout)
		if err != nil {
			jww.ERROR.Printf(err.Error())
			continue
		}
		newParams[storage.AdvertisementTimeout] = realtimeDelay
		valueStr, err := storage.PermissioningDb.GetStateValue(storage.PoolThreshold)
		if err != nil {
			jww.ERROR.Printf("Unable to find %s: %+v", storage.PoolThreshold, err)
			continue
		}
		threshold, err := strconv.ParseFloat(valueStr, 64)
		if err != nil {
			jww.ERROR.Printf("Unable to decode %s: %+v", valueStr, err)
			continue
		}

		jww.INFO.Printf("Preparing to update scheduling params...")
		params.Lock()
		jww.INFO.Printf("Updating scheduling params: %+v, %s: %f", newParams, storage.PoolThreshold, threshold)
		params.TeamSize = uint32(teamSize)
		params.BatchSize = uint32(batchSize)
		params.PrecomputationTimeout = time.Duration(precompTimeout)
		params.RealtimeTimeout = time.Duration(realtimeTimeout)
		params.MinimumDelay = time.Duration(minDelay)
		params.RealtimeDelay = time.Duration(realtimeDelay)
		params.Threshold = threshold
		params.Unlock()

		time.Sleep(updateFreq)
	}

}

// Scheduler is a utility function which builds a round by handling a node's
// state changes then creating a team from the nodes in the pool
func Scheduler(params *SafeParams, state *storage.NetworkState, killchan chan chan struct{}) error {

	rng := fastRNG.NewStreamGenerator(10000,
		uint(runtime.NumCPU()), csprng.NewSystemRNG)

	// Pool which tracks nodes which are not in a team
	pool := NewWaitingPool()

	// Channel to send new rounds over to be created
	newRoundChan := make(chan protoRound, newRoundChanLen)

	// Select the correct round creator
	var createRound roundCreator

	// Set teaming algorithm
	jww.INFO.Printf("Using Secure Teaming Algorithm")
	createRound = createSecureRound

	// Channel to communicate that a round has timed out
	roundTimeoutTracker := make(chan id.Round, 1000)

	roundTracker := NewRoundTracker()

	//begin the thread that starts rounds
	go func() {

		lastRound := time.Now()

		paramsCopy := params.SafeCopy()
		minRoundDelay := (paramsCopy.MinimumDelay * time.Millisecond) / 3
		var err error
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
				paramsCopy.PrecomputationTimeout*time.Millisecond,
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

	paramsCopy := params.SafeCopy()

	sc := &stateChanger{
		lastRealtime:     time.Unix(0, 0),
		realtimeDelay:    paramsCopy.RealtimeDelay * time.Millisecond,
		realtimeDelta:    paramsCopy.MinimumDelay * time.Millisecond,
		realtimeTimeout:  paramsCopy.RealtimeTimeout * time.Millisecond,
		pool:             pool,
		state:            state,
		roundTracker:     roundTracker,
		roundTimeoutChan: roundTimeoutTracker,
	}

	jww.INFO.Printf("Initialized state changer with: "+
		"\n\t realtimeDelay: %s, "+
		"\n\t realtimeDelta: %s"+
		"\n\t realtimeTimeout: %s", sc.realtimeDelay,
		sc.realtimeDelta, sc.realtimeTimeout)

	// Start receiving updates from nodes
	for true {

		isRoundTimeout := false
		var update node.UpdateNotification
		var timedOutRoundID id.Round
		hasUpdate := false

		select {
		// Receive a signal to kill the Scheduler
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
			err = sc.HandleNodeUpdates(update)
			if err != nil {
				return err
			}
		}

		for {
			//get the pool of disabled nodes and determine how many
			//nodes can be scheduled
			numNodesInPool := pool.Len()

			// Create a new round if the pool is full
			var teamFormationThreshold int
			teamSize := int(paramsCopy.TeamSize)
			teamFormationThreshold = int(paramsCopy.Threshold * float64(state.CountActiveNodes()))
			if numNodesInPool >= teamFormationThreshold && numNodesInPool >= teamSize && killed == nil {

				// Increment round ID
				currentID, err := state.IncrementRoundID()

				if err != nil {
					return err
				}

				stream := rng.GetStream()
				newRound, err := createRound(paramsCopy, pool, teamFormationThreshold, currentID, state, stream)
				stream.Close()
				if err != nil {
					return err
				}
				// Send the round to the new round channel to be created
				newRoundChan <- newRound
			} else {
				break
			}
		}

		// If the Scheduler is to be killed and no rounds are in progress,
		// kill the Scheduler
		if killed != nil && roundTracker.Len() == 0 {
			// Stop round creation
			close(newRoundChan)
			jww.WARN.Printf("Scheduler is exiting due to kill signal")
			killed <- struct{}{}
			return nil
		}
	}

	return errors.New("single Scheduler should never exit")
}

// Helper function which handles when we receive a timed out round
func timeoutRound(state *storage.NetworkState, timeoutRoundID id.Round,
	roundTracker *RoundTracker) error {
	// On a timeout, check if the round is completed. If not, kill it
	ourRound, exists := state.GetRoundMap().GetRound(timeoutRoundID)
	if !exists {
		jww.ERROR.Printf("Failed to timeout round - round not found. " +
			"This is a rare race condition, if seen extremely rarely this " +
			"is not a problem")
	}
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
