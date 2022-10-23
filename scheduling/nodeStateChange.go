////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package scheduling

// Contains the handler for node updates
import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// HandleNodeUpdates handles the node state changes.
//  A node in waiting is added to the pool in preparation for precomputing.
//  A node in standby is added to a round in preparation for realtime.
//  A node in completed waits for all other nodes in the team to transition
//   before the round is updated.
func HandleNodeUpdates(update node.UpdateNotification, pool *waitingPool, state *storage.NetworkState,
	realtimeDelay, realtimeDelta time.Duration, roundTracker *RoundTracker, roundTimeoutChan chan id.Round,
	realtimeTimeout time.Duration, lastRealtime *time.Time) error {
	// Check the round's error state
	n := state.GetNodeMap().GetNode(update.Node)
	// when a node poll is received, the nodes polling lock is taken.  If there
	// is no update, it is released in the endpoint, otherwise it is released
	// here which blocks all future polls until processing completes
	defer n.GetPollingLock().Unlock()
	hasRound, r := n.GetCurrentRound()

	// Enforce that only error updates are allowed for a failed round
	roundErrored := hasRound == true && r.GetRoundState() == states.FAILED && update.ToActivity != current.ERROR
	if roundErrored {
		jww.WARN.Printf("Round %d has failed, state for %s cannot be updated to %s, moving to %s",
			r.GetRoundID(), update.Node.String(), update.ToActivity.String(), current.ERROR)
		update.ToActivity = current.ERROR
	}

	if update.ClientErrors != nil && len(update.ClientErrors) > 0 {
		r.AppendClientErrors(update.ClientErrors)
	}
	//ban the node if it is supposed to be banned
	if update.ToStatus == node.Banned {
		if hasRound {
			banError := &pb.RoundError{
				Id:     uint64(r.GetRoundID()),
				NodeId: id.Permissioning.Marshal(),
				Error:  fmt.Sprintf("Round killed due to particiption of banned node %s", update.Node),
			}
			err := signature.SignRsa(banError, state.GetPrivateKey())
			if err != nil {
				return errors.Errorf("Failed to sign error message for banned node %s: %+v", update.Node, err)
			}
			n.ClearRound()
			return killRound(state, r, banError, roundTracker)
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
		// If the node was in the offline pool, set it to online
		//  (which also adds it to the online pool)
		if update.FromStatus == node.Inactive && update.ToStatus == node.Active {
			pool.SetNodeToOnline(n)
		} else {
			// Otherwise, add it to the online pool normally
			pool.Add(n)
		}

	case current.PRECOMPUTING:
		// Check that node in precomputing does have a round
		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be moving to the %s state", update.Node, states.PRECOMPUTING)
		}

	case current.STANDBY:
		// Check that node in standby actually does have a round
		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be in %s state", update.Node, states.PRECOMPUTING)
		}
		// Check if the round is ready for all the nodes
		// in order to transition
		stateComplete := r.NodeIsReadyForTransition()
		if stateComplete {
			// Update the round for end of precomp transition
			err := r.Update(states.STANDBY, time.Now())

			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.PRECOMPUTING, states.STANDBY)
			}

			// kill the precomp timeout and start a realtime timeout
			r.DenoteRoundCompleted()
			go waitForRoundTimeout(roundTimeoutChan, state, r,
				realtimeTimeout, "realtime")

			startTime := time.Now().Add(realtimeDelay)
			if (*lastRealtime).Sub(startTime) < realtimeDelta {
				startTime = (*lastRealtime).Add(realtimeDelta)
			}

			lastRealtime = &startTime

			// Update the round for realtime transition
			err = r.Update(states.QUEUED, startTime)

			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.STANDBY, states.QUEUED)
			}

			// Build the round info and add to the networkState
			err = state.AddRoundUpdate(r.BuildRoundInfo())
			if err != nil {
				return errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.STANDBY, states.QUEUED)
			}

		}
	case current.REALTIME:
		// Check that node in standby actually does have a round
		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be moving to the %s state", update.Node, states.REALTIME)
		}
		// REALTIME does not use the state complete handler because it
		// increments on the first report, not when every node reports in
		// order to avoid distributed synchronicity issues
		if r.GetRoundState() != states.REALTIME {

			err := r.Update(states.REALTIME, time.Now())

			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.QUEUED, states.REALTIME)
			}
		}
	case current.COMPLETED:
		// Check that node in standby actually does have a round
		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be in %s state", update.Node, states.COMPLETED)
		}

		// Clear the round
		n.ClearRound()

		// Keep track of when the first node reached the completed state
		if r.GetTopology().IsLastNode(n.GetID()) {
			r.SetRealtimeCompletedTs(time.Now().UnixNano())
		}

		// Check if the round is ready for all the nodes
		// in order to transition
		stateComplete := r.NodeIsReadyForTransition()

		if stateComplete {
			// Update the round for realtime transition
			err := r.Update(states.COMPLETED, time.Now())
			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}

			// Build the round info and add to the networkState
			roundInfo := r.BuildRoundInfo()
			err = state.AddRoundUpdate(roundInfo)
			if err != nil {
				return errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}

			//send the signal that the round is complete
			r.DenoteRoundCompleted()
			roundTracker.RemoveActiveRound(r.GetRoundID())

			// Store round metric in another thread for completed round
			go StoreRoundMetric(roundInfo, r.GetRoundState(), r.GetRealtimeCompletedTs())

			// Commit metrics about the round to storage
			return nil
		}
	case current.ERROR:

		// If in an error state, kill the round if the node has one
		var err error
		if hasRound {
			//send the signal that the round is complete
			r.DenoteRoundCompleted()
			n.ClearRound()
			err = killRound(state, r, update.Error, roundTracker)
		}
		return err
	}

	return nil
}

// Insert metrics about the newly-completed round into storage
func StoreRoundMetric(roundInfo *pb.RoundInfo, roundEnd states.Round, realtimeTs int64) {
	metric := &storage.RoundMetric{
		Id:            roundInfo.ID,
		PrecompStart:  time.Unix(0, int64(roundInfo.Timestamps[states.PRECOMPUTING])),
		PrecompEnd:    time.Unix(0, int64(roundInfo.Timestamps[states.STANDBY])),
		RealtimeStart: time.Unix(0, int64(roundInfo.Timestamps[states.REALTIME])),
		RealtimeEnd:   time.Unix(0, realtimeTs),
		RoundEnd:      time.Unix(0, int64(roundInfo.Timestamps[roundEnd])),
		BatchSize:     roundInfo.BatchSize,
	}

	precompDuration := metric.PrecompEnd.Sub(metric.PrecompStart)
	realTimeDuration := metric.RealtimeEnd.Sub(metric.RealtimeStart)

	jww.TRACE.Printf("Precomp for round %v took: %v", roundInfo.GetRoundId(), precompDuration)
	jww.TRACE.Printf("Realtime for round %v took: %v", roundInfo.GetRoundId(), realTimeDuration)

	err := storage.PermissioningDb.InsertRoundMetric(metric, roundInfo.Topology)
	if err != nil {
		jww.ERROR.Printf("Failed to insert metric for round %d: %+v",
			roundInfo.GetRoundId(), err)
	}
}

// killRound sets the round to failed and clears the node's round
func killRound(state *storage.NetworkState, r *round.State,
	roundError *pb.RoundError, roundTracker *RoundTracker) error {

	roundId := r.GetRoundID()
	r.AppendError(roundError)

	err := r.Update(states.FAILED, time.Now())
	if err == nil {
		roundTracker.RemoveActiveRound(roundId)
	}

	roundInfo := r.BuildRoundInfo()

	// Build the round info and update the network state
	err = state.AddRoundUpdate(roundInfo)
	if err != nil {
		return errors.WithMessagef(err, "Could not issue "+
			"update to kill round %v", roundId)
	}

	go func() {
		// Attempt to insert the RoundMetric for the failed round
		StoreRoundMetric(roundInfo, r.GetRoundState(), 0)

		// Return early if there is no roundError
		if roundError == nil {
			return
		}

		nid, err := id.Unmarshal(roundError.NodeId)
		var idStr string
		if err != nil {
			idStr = "N/A"
		} else {
			idStr = nid.String()
		}

		formattedError := fmt.Sprintf("Round Error from %s: %s", idStr, roundError.Error)
		jww.INFO.Print(formattedError)

		// Next, attempt to insert the error for the failed round
		err = storage.PermissioningDb.InsertRoundError(roundId, formattedError)
		if err != nil {
			jww.WARN.Printf("Could not insert round error: %+v", err)
		}
	}()

	return nil
}
