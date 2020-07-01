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
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
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

	// If resource queue timeout isn't set, set it to a default of 3 minutes
	if params.ResourceQueueTimeout == 0 {
		params.ResourceQueueTimeout = 180000 // 180000 ms = 3 minutes
	}
	// If roundTimeout hasn't set, set to a default of one minute
	if params.RoundTimeout == 0 {
		params.RoundTimeout = 60
	}

	return scheduler(params, state, killchan)
}

// scheduler is a utility function which builds a round by handling a node's
// state changes then creating a team from the nodes in the pool
func scheduler(params Params, state *storage.NetworkState, killchan chan chan struct{}) error {

	// Pool which tracks nodes which are not in a team
	pool := NewWaitingPool()

	// Channel to send new rounds over to be created
	newRoundChan := make(chan protoRound, newRoundChanLen)

	// Calculate the realtime delay from params
	rtDelay := params.RealtimeDelay * time.Millisecond

	// Select the correct round creator
	var createRound roundCreator
	var teamFormationThreshold uint32

	// Identify which teaming algorithm we will be using
	if params.Secure {
		jww.INFO.Printf("Using Secure Teaming Algorithm")
		createRound = createSecureRound
		teamFormationThreshold = params.Threshold
	} else {
		jww.INFO.Printf("Using Simple Teaming Algorithm")
		createRound = createSimpleRound
		teamFormationThreshold = params.TeamSize
	}

	// Channel to communicate that a round has timed out
	roundTimeoutTracker := make(chan id.Round, 1000)

	roundTracker := NewRoundTracker()

	//begin the thread that starts rounds
	go func() {
		lastRound := time.Now()
		var err error
		minRoundDelay := params.MinimumDelay * time.Millisecond
		for newRound := range newRoundChan {

			// To avoid back-to-back teaming, we make sure to sleep until the minimum delay
			if timeDiff := time.Now().Sub(lastRound); timeDiff < minRoundDelay {
				time.Sleep(minRoundDelay - timeDiff)
			}
			lastRound = time.Now()

			err = startRound(newRound, state, roundTracker)
			if err != nil {
				break
			}

			go func(roundID id.Round) {
				// Allow for round the to be added to the map
				ourRound := state.GetRoundMap().GetRound(roundID)
				roundTimer := time.NewTimer(params.RoundTimeout * time.Second)
				select {
				// Wait for the timer to go off
				case <-roundTimer.C:

					// Send the timed out round id to the timeout handler
					jww.INFO.Printf("Round %v has timed out, signaling exit", roundID)
					roundTimeoutTracker <- roundID
				// Signals the round has been completed.
				// In this case, we can exit the go-routine
				case <-ourRound.GetRoundCompletedChan():
					return
				}
			}(newRound.ID)
		}

		jww.ERROR.Printf("Round creation thread should never exit: %s", err)

	}()

	var killed chan struct{}

	numRounds := 0

	// optional debug print which regularly prints the status of rounds and nodes
	// turned on by setting DebugTrackRounds to true in the scheduling config
	if params.DebugTrackRounds {
		go trackRounds(params, state, pool, roundTracker)
	}

	// Start receiving updates from nodes
	for true {
		isRoundTimeout := false
		var update node.UpdateNotification
		var timedOutRoundID id.Round
		select {
		// Receive a signal to kill the scheduler
		case killed = <-killchan:
		// When we get a node update, move past the select statement
		case update = <-state.GetNodeUpdateChannel():
		// Receive a signal indicating that a round has timed out
		case timedOutRoundID = <-roundTimeoutTracker:
			isRoundTimeout = true
		}

		endRound := false

		if isRoundTimeout {
			// Handle the timed out round
			err := timeoutRound(state, timedOutRoundID, roundTracker)
			if err != nil {
				return err
			}
			endRound = true
		} else {
			var err error
			// Handle the node's state change
			endRound, err = HandleNodeUpdates(update, pool, state,
				rtDelay, roundTracker)
			if err != nil {
				return err
			}
		}

		// Remove offline nodes from pool to more accurately determine if pool is eligible for round creation
		pool.CleanOfflineNodes(params.NodeCleanUpInterval * time.Second)

		// If a round has finished, decrement num rounds
		if endRound {
			numRounds--
		}

		for {
			// Create a new round if the pool is full
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

				// Send the round to the new round channel to be created
				newRoundChan <- newRound
				numRounds++
			} else {
				break
			}
		}

		// If the scheduler is to be killed and no rounds are in progress,
		// kill the scheduler
		if killed != nil && numRounds == 0 {
			close(newRoundChan)
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
		// Build the round error message
		timeoutError := &pb.RoundError{
			Id:     uint64(ourRound.GetRoundID()),
			NodeId: id.Permissioning.Marshal(),
			Error:  fmt.Sprintf("Round %d killed due to a round time out", ourRound.GetRoundID()),
		}
		// Sign the error message with our private key
		err := signature.Sign(timeoutError, state.GetPrivateKey())
		if err != nil {
			jww.FATAL.Panicf("Failed to sign error message for "+
				"timed out round %d: %+v", ourRound.GetRoundID(), err)
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

// Tracks rounds, periodically outputs how many teams are in various rounds
func trackRounds(params Params, state *storage.NetworkState, pool *waitingPool,
	roundTracker *RoundTracker) {
	// Period of polling the state map for logs
	schedulingTicker := time.NewTicker(1 * time.Minute)

	for true {
		realtimeNodes := 0
		precompNodes := 0
		waitingNodes := 0
		noPoll := make([]string, 0)
		notUpdating := make([]string, 0)
		goodNode := make([]string, 0)
		noContact := make([]string, 0)

		precompRounds := make([]*round.State, 0)
		queuedRounds := make([]*round.State, 0)
		realtimeRounds := make([]*round.State, 0)
		otherRounds := make([]*round.State, 0)

		<-schedulingTicker.C
		now := time.Now()

		// Parse through the node map to collect nodes into round state arrays
		nodeStates := state.GetNodeMap().GetNodeStates()

		for _, nodeState := range nodeStates {
			switch nodeState.GetActivity() {
			case current.WAITING:
				waitingNodes++
			case current.REALTIME:
				realtimeNodes++
			case current.PRECOMPUTING:
				precompNodes++
			}

			//tracks which nodes have not acted recently
			lastPoll := nodeState.GetLastPoll()
			lastUpdate := nodeState.GetLastUpdate()
			pollDelta := time.Duration(0)
			updateDelta := time.Duration(0)

			if now.After(lastPoll){
				pollDelta = now.Sub(lastPoll)
			}

			if now.After(lastUpdate){
				updateDelta = now.Sub(lastUpdate)
			}

			if pollDelta > timeToInactive {
				s := fmt.Sprintf("\tNode %s (AppID: %v, Activity: %s) has not polled for %s", nodeState.GetID(), nodeState.GetAppID(), nodeState.GetActivity(), lastPoll)
				noPoll = append(noPoll, s)
			}else if updateDelta > timeToInactive {
				s := fmt.Sprintf("\tNode %s (AppID: %v) stuck in %s for %s (last poll: %s)", nodeState.GetID(), nodeState.GetAppID(), nodeState.GetActivity(), updateDelta, pollDelta)
				notUpdating = append(notUpdating, s)
			}else{
				s := fmt.Sprintf("\tNode %s (AppID: %v) operating correctly in %s for %s (last poll: %s)", nodeState.GetID(), nodeState.GetAppID(), nodeState.GetActivity(), updateDelta, pollDelta)
				goodNode = append(goodNode, s)
			}

			//tracks if the node cannot be contacted by permissioning
			if nodeState.GetRawConnectivity() == node.PortFailed {
				s := fmt.Sprintf("\tNode %s (AppID: %v, Activity: %s) cannot be contacted", nodeState.GetID(), nodeState.GetAppID(), nodeState.GetActivity())
				noContact = append(noContact, s)
			}
		}
		// Parse through the active round list to collect into round state arrays
		rounds := roundTracker.GetActiveRounds()

		for _, rid := range rounds {
			r := state.GetRoundMap().GetRound(rid)
			switch r.GetRoundState() {
			case states.PRECOMPUTING:
				precompRounds = append(precompRounds, r)
			case states.QUEUED:
				queuedRounds = append(queuedRounds, r)
			case states.REALTIME:
				realtimeRounds = append(realtimeRounds, r)
			default:
				otherRounds = append(otherRounds, r)
			}
		}

		// Output data into logs
		jww.INFO.Printf("Teams in precomp: %v", len(precompRounds))
		jww.INFO.Printf("Teams in queued: %v", len(queuedRounds))
		jww.INFO.Printf("Teams in realtime: %v", len(realtimeRounds))
		jww.INFO.Printf("")
		jww.INFO.Printf("Nodes in waiting: %v", waitingNodes)
		jww.INFO.Printf("Nodes in precomp: %v", precompNodes)
		jww.INFO.Printf("Nodes in realtime: %v", realtimeNodes)
		jww.INFO.Printf("")
		jww.INFO.Printf("Nodes in pool: %v", pool.Len())
		jww.INFO.Printf("Nodes in offline pool: %v", pool.OfflineLen())
		jww.INFO.Printf("")
		jww.INFO.Printf("Total Nodes: %v", len(nodeStates))
		jww.INFO.Printf("Nodes without recent poll: %v", len(noPoll))
		jww.INFO.Printf("Nodes without recent update: %v", len(notUpdating))
		jww.INFO.Printf("Normally operating nodes: %v", len(nodeStates) - len(noPoll) - len(notUpdating))
		jww.INFO.Printf("")


		if len(goodNode) > 0 {
			jww.INFO.Printf("Nodes which operating as expected")
			for _, s := range goodNode{
				jww.INFO.Print(s)
			}
			jww.INFO.Printf("")
		}
		if len(noPoll) > 0 {
			jww.INFO.Printf("Nodes with no polls in: %s", timeToInactive)
			for _, s := range noPoll{
				jww.INFO.Print(s)
			}
			jww.INFO.Printf("")
		}

		if len(notUpdating) > 0 {
			jww.INFO.Printf("Nodes with no state updates in: %s", timeToInactive)
			for _, s := range notUpdating{
				jww.INFO.Print(s)
			}
			jww.INFO.Printf("")
		}

		if len(noContact) > 0 {
			jww.INFO.Printf("Nodes which are not included due to no contact error")
			for _, s := range noContact{
				jww.INFO.Print(s)
			}
			jww.INFO.Printf("")
		}

		allRounds := precompRounds
		allRounds = append(allRounds, queuedRounds...)
		allRounds = append(allRounds, realtimeRounds...)
		allRounds = append(allRounds, otherRounds...)
		jww.INFO.Printf("All Active Rounds")
		if len(allRounds) > 0 {
			for _, r := range allRounds {
				lastUpdate := r.GetLastUpdate()
				var delta time.Duration
				if lastUpdate.After(now) {
					delta = 0
				} else {
					delta = now.Sub(lastUpdate)
				}
				jww.INFO.Printf("\tRound %v in state %s, last update: %s ago", r.GetRoundID(), r.GetRoundState(), delta)
			}
		} else {
			jww.INFO.Printf("\tNo Rounds active")
		}
		jww.INFO.Printf("")
	}
}
