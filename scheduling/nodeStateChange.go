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
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"time"
)

// HandleNodeUpdates handles the node state changes.
//  A node in waiting is added to the pool in preparation for precomputing.
//  A node in standby is added to a round in preparation for realtime.
//  A node in completed waits for all other nodes in the team to transition
//   before the round is updated.
func HandleNodeUpdates(update node.UpdateNotification, pool *waitingPool,
	state *storage.NetworkState, realtimeDelay time.Duration) (bool, error) {
	// Check the round's error state
	n := state.GetNodeMap().GetNode(update.Node)
	// when a node poll is received, the nodes polling lock is taken.  If there
	// is no update, it is released in the endpoint, otherwise it is released
	// here which blocks all future polls until processing completes
	defer n.GetPollingLock().Unlock()
	hasRound, r := n.GetCurrentRound()
	roundErrored := hasRound == true && r.GetRoundState() == states.FAILED && update.ToActivity != current.ERROR
	if roundErrored {
		return false, nil
	}
	//ban the node if it is supposed to be banned
	if update.ToStatus == node.Banned {
		if hasRound {
			banError := &pb.RoundError{
				Id:     uint64(r.GetRoundID()),
				NodeId: id.Permissioning.Marshal(),
				Error:  fmt.Sprintf("Round killed due to particiption of banned node %s", update.Node),
			}
			err := signature.Sign(banError, state.GetPrivateKey())
			if err != nil {
				jww.FATAL.Panicf("Failed to sign error message for banned node %s: %+v", update.Node, err)
			}
			return false, killRound(state, r, n, banError)
		} else {
			pool.Ban(n)
			return false, nil
		}
	}
	//get node and round information
	switch update.ToActivity {
	case current.NOT_STARTED:
		// Do nothing
	case current.WAITING:
		// Clear the round if node has one (it should unless it is
		// coming from NOT_STARTED
		if hasRound {
			n.ClearRound()
		}
		pool.Add(n)
	case current.PRECOMPUTING:
		// Check that node in precomputing does have a round
		if !hasRound {
			return false, errors.Errorf("Node %s without round should "+
				"not be moving to the %s state", update.Node, states.PRECOMPUTING)
		}
		// fixme: nodes selected from pool are assigned to precomp in start round, inherently are synced
		//stateComplete := r.NodeIsReadyForTransition()
		//if stateComplete {
		//	err := r.Update(states.PRECOMPUTING, time.Now())
		//	if err != nil {
		//		return errors.WithMessagef(err,
		//			"Could not move round %v from %s to %s",
		//			r.GetRoundID(), states.PENDING, states.PRECOMPUTING)
		//	}
		//}
	case current.STANDBY:
		// Check that node in standby actually does have a round
		if !hasRound {
			return false, errors.Errorf("Node %s without round should "+
				"not be in %s state", update.Node, states.PRECOMPUTING)
		}
		// Check if the round is ready for all the nodes
		// in order to transition
		stateComplete := r.NodeIsReadyForTransition()
		if stateComplete {
			// Update the round for realtime transition
			err := r.Update(states.QUEUED, time.Now().Add(realtimeDelay))
			if err != nil {
				return false, errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.PRECOMPUTING, states.REALTIME)
			}
			// Build the round info and add to the networkState
			err = state.AddRoundUpdate(r.BuildRoundInfo())
			if err != nil {
				return false, errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.PRECOMPUTING, states.REALTIME)
			}
		}
	case current.REALTIME:
		// Check that node in standby actually does have a round
		if !hasRound {
			return false, errors.Errorf("Node %s without round should "+
				"not be moving to the %s state", update.Node, states.REALTIME)
		}
		stateComplete := r.NodeIsReadyForTransition()
		if stateComplete {
			err := r.Update(states.REALTIME, time.Now())
			if err != nil {
				return false, errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.STANDBY, states.REALTIME)
			}
		}
	case current.COMPLETED:
		// Check that node in standby actually does have a round
		if !hasRound {
			return false, errors.Errorf("Node %s without round should "+
				"not be in %s state", update.Node, states.COMPLETED)
		}
		// Clear the round
		n.ClearRound()
		// Check if the round is ready for all the nodes
		// in order to transition
		stateComplete := r.NodeIsReadyForTransition()
		if stateComplete {
			// Update the round for realtime transition
			err := r.Update(states.COMPLETED, time.Now())
			if err != nil {
				return false, errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}
			// Build the round info and add to the networkState
			roundInfo := r.BuildRoundInfo()
			err = state.AddRoundUpdate(roundInfo)
			if err != nil {
				return false, errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}
			// Commit metrics about the round to storage
			return true, StoreRoundMetric(roundInfo)
		}
	case current.ERROR:
		// If in an error state, kill the round
		return false, killRound(state, r, n, update.Error)
	}

	return false, nil
}

// Insert metrics about the newly-completed round into storage
func StoreRoundMetric(roundInfo *pb.RoundInfo) error {
	metric := &storage.RoundMetric{
		PrecompStart:  time.Unix(0, int64(roundInfo.Timestamps[states.PRECOMPUTING])),
		PrecompEnd:    time.Unix(0, int64(roundInfo.Timestamps[states.STANDBY])),
		RealtimeStart: time.Unix(0, int64(roundInfo.Timestamps[states.REALTIME])),
		RealtimeEnd:   time.Unix(0, int64(roundInfo.Timestamps[states.COMPLETED])),
		BatchSize:     roundInfo.BatchSize,
	}

	precompDuration := metric.PrecompEnd.Sub(metric.PrecompStart)
	realTimeDuration := metric.RealtimeEnd.Sub(metric.RealtimeStart)

	jww.TRACE.Printf("Precomp for round %v took: %v", roundInfo.GetRoundId(), precompDuration)
	jww.TRACE.Printf("Realtime for round %v took: %v", roundInfo.GetRoundId(), realTimeDuration)

	return storage.PermissioningDb.InsertRoundMetric(metric, roundInfo.Topology)
}

// killRound sets the round to failed and clears the node's round
func killRound(state *storage.NetworkState, r *round.State, n *node.State, roundError *pb.RoundError) error {

	r.AppendError(roundError)
	_ = r.Update(states.FAILED, time.Now())
	n.ClearRound()
	roundId := r.GetRoundID()

	// Build the round info and update the network state
	err := state.AddRoundUpdate(r.BuildRoundInfo())
	if err != nil {
		return errors.WithMessagef(err, "Could not issue "+
			"update to kill round %v", r.GetRoundID())
	}

	// Attempt to insert the RoundMetric for the failed round
	metric := &storage.RoundMetric{
		Id:            uint64(roundId),
		PrecompStart:  time.Unix(0, int64(r.BuildRoundInfo().Timestamps[states.PRECOMPUTING])),
		PrecompEnd:    time.Unix(0, int64(r.BuildRoundInfo().Timestamps[states.STANDBY])),
		RealtimeStart: time.Unix(0, int64(r.BuildRoundInfo().Timestamps[states.REALTIME])),
		RealtimeEnd:   time.Unix(0, int64(r.BuildRoundInfo().Timestamps[states.FAILED])),
		BatchSize:     r.BuildRoundInfo().BatchSize,
	}
	err = storage.PermissioningDb.InsertRoundMetric(metric,
		r.BuildRoundInfo().Topology)
	if err != nil {
		return errors.WithMessagef(err, "Could not insert round metric: %+v", err)
	}

	// Next, attempt to insert the error for the failed round
	err = storage.PermissioningDb.InsertRoundError(roundId, roundError.Error)
	if err != nil {
		err = errors.WithMessagef(err, "Could not insert round error: %+v", err)
	}
	return err
}
