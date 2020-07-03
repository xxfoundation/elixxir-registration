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
	"sync/atomic"
	"time"
)

// HandleNodeUpdates handles the node state changes.
//  A node in waiting is added to the pool in preparation for precomputing.
//  A node in standby is added to a round in preparation for realtime.
//  A node in completed waits for all other nodes in the team to transition
//   before the round is updated.
func HandleNodeUpdates(update node.UpdateNotification, pool *waitingPool, state *storage.NetworkState,
	realtimeDelay time.Duration, roundTracker *RoundTracker) error {
	// Check the round's error state
	atomic.StoreUint32(scheduleTracker, 30)
	n := state.GetNodeMap().GetNode(update.Node)
	atomic.StoreUint32(scheduleTracker, 31)
	// when a node poll is received, the nodes polling lock is taken.  If there
	// is no update, it is released in the endpoint, otherwise it is released
	// here which blocks all future polls until processing completes
	defer n.GetPollingLock().Unlock()
	atomic.StoreUint32(scheduleTracker, 32)
	hasRound, r := n.GetCurrentRound()
	atomic.StoreUint32(scheduleTracker, 33)
	roundErrored := hasRound == true && r.GetRoundState() == states.FAILED && update.ToActivity != current.ERROR
	if roundErrored {
		return nil
	}
	atomic.StoreUint32(scheduleTracker, 34)
	//ban the node if it is supposed to be banned
	if update.ToStatus == node.Banned {
		atomic.StoreUint32(scheduleTracker, 35)
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
			atomic.StoreUint32(scheduleTracker, 36)
			n.ClearRound()
			atomic.StoreUint32(scheduleTracker, 37)
			return killRound(state, r, banError, roundTracker, pool)
		} else {
			atomic.StoreUint32(scheduleTracker, 38)
			pool.Ban(n)
			atomic.StoreUint32(scheduleTracker, 39)
			return nil
		}
	}

	atomic.StoreUint32(scheduleTracker, 40)
	//get node and round information
	switch update.ToActivity {
	case current.NOT_STARTED:
		atomic.StoreUint32(scheduleTracker, 41)
		// Do nothing
	case current.WAITING:
		atomic.StoreUint32(scheduleTracker, 42)
		// If the node was in the offline pool, set it to online
		//  (which also adds it to the online pool)
		if update.FromStatus == node.Inactive && update.ToStatus == node.Active {
			atomic.StoreUint32(scheduleTracker, 43)
			pool.SetNodeToOnline(n)
		} else {
			atomic.StoreUint32(scheduleTracker, 44)
			// Otherwise, add it to the online pool normally
			pool.Add(n)
		}
		atomic.StoreUint32(scheduleTracker, 45)

	case current.PRECOMPUTING:
		atomic.StoreUint32(scheduleTracker, 46)
		// Check that node in precomputing does have a round
		if !hasRound {
			atomic.StoreUint32(scheduleTracker, 47)
			return errors.Errorf("Node %s without round should "+
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
		atomic.StoreUint32(scheduleTracker, 48)
		// Check that node in standby actually does have a round
		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be in %s state", update.Node, states.PRECOMPUTING)
		}
		atomic.StoreUint32(scheduleTracker, 49)
		// Check if the round is ready for all the nodes
		// in order to transition
		stateComplete := r.NodeIsReadyForTransition()
		atomic.StoreUint32(scheduleTracker, 50)
		if stateComplete {
			atomic.StoreUint32(scheduleTracker, 51)
			// Update the round for end of precomp transition
			err := r.Update(states.STANDBY, time.Now())
			atomic.StoreUint32(scheduleTracker, 52)
			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.PRECOMPUTING, states.STANDBY)
			}
			atomic.StoreUint32(scheduleTracker, 53)
			// Update the round for realtime transition
			err = r.Update(states.QUEUED, time.Now().Add(realtimeDelay))
			atomic.StoreUint32(scheduleTracker, 54)
			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.STANDBY, states.QUEUED)
			}
			atomic.StoreUint32(scheduleTracker, 55)
			// Build the round info and add to the networkState
			err = state.AddRoundUpdate(r.BuildRoundInfo())
			atomic.StoreUint32(scheduleTracker, 56)
			if err != nil {
				return errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.STANDBY, states.QUEUED)
			}
			atomic.StoreUint32(scheduleTracker, 57)
		}
	case current.REALTIME:
		// Check that node in standby actually does have a round
		atomic.StoreUint32(scheduleTracker, 58)
		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be moving to the %s state", update.Node, states.REALTIME)
		}
		// REALTIME does not use the state complete handler because it
		// increments on the first report, not when every node reports in
		// order to avoid distributed synchronicity issues
		if r.GetRoundState() != states.REALTIME {
			atomic.StoreUint32(scheduleTracker, 59)
			err := r.Update(states.REALTIME, time.Now())
			atomic.StoreUint32(scheduleTracker, 60)
			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.QUEUED, states.REALTIME)
			}
			atomic.StoreUint32(scheduleTracker, 61)
		}
		atomic.StoreUint32(scheduleTracker, 61)
	case current.COMPLETED:
		atomic.StoreUint32(scheduleTracker, 62)
		// Check that node in standby actually does have a round
		if !hasRound {
			return errors.Errorf("Node %s without round should "+
				"not be in %s state", update.Node, states.COMPLETED)
		}
		atomic.StoreUint32(scheduleTracker, 63)
		// Clear the round
		n.ClearRound()
		atomic.StoreUint32(scheduleTracker, 64)
		// Check if the round is ready for all the nodes
		// in order to transition
		stateComplete := r.NodeIsReadyForTransition()
		atomic.StoreUint32(scheduleTracker, 65)
		if stateComplete {
			// Update the round for realtime transition
			atomic.StoreUint32(scheduleTracker, 66)
			err := r.Update(states.COMPLETED, time.Now())
			atomic.StoreUint32(scheduleTracker, 67)
			if err != nil {
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}
			atomic.StoreUint32(scheduleTracker, 68)

			// Build the round info and add to the networkState
			roundInfo := r.BuildRoundInfo()
			atomic.StoreUint32(scheduleTracker, 69)
			err = state.AddRoundUpdate(roundInfo)
			atomic.StoreUint32(scheduleTracker, 70)
			if err != nil {
				return errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}
			atomic.StoreUint32(scheduleTracker, 71)

			//send the signal that the round is complete
			r.DenoteRoundCompleted()
			atomic.StoreUint32(scheduleTracker, 472)
			roundTracker.RemoveActiveRound(r.GetRoundID())
			atomic.StoreUint32(scheduleTracker, 73)
			// Commit metrics about the round to storage
			return StoreRoundMetric(roundInfo)
		}
	case current.ERROR:
		atomic.StoreUint32(scheduleTracker, 74)
		// If in an error state, kill the round if the node has one
		var err error
		atomic.StoreUint32(scheduleTracker, 75)
		if hasRound {
			atomic.StoreUint32(scheduleTracker, 76)
			//send the signal that the round is complete
			r.DenoteRoundCompleted()
			atomic.StoreUint32(scheduleTracker, 77)
			n.ClearRound()
			atomic.StoreUint32(scheduleTracker, 78)
			err = killRound(state, r, update.Error, roundTracker, pool)
			atomic.StoreUint32(scheduleTracker, 79)
		}
		atomic.StoreUint32(scheduleTracker, 80)
		return err
	}

	return nil
}

// Insert metrics about the newly-completed round into storage
func StoreRoundMetric(roundInfo *pb.RoundInfo) error {
	metric := &storage.RoundMetric{
		Id:            roundInfo.ID,
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
func killRound(state *storage.NetworkState, r *round.State,
	roundError *pb.RoundError, roundTracker *RoundTracker, pool *waitingPool) error {
	atomic.StoreUint32(scheduleTracker, 16)
	r.AppendError(roundError)
	atomic.StoreUint32(scheduleTracker, 17)
	err := r.Update(states.FAILED, time.Now())
	atomic.StoreUint32(scheduleTracker, 18)
	if err == nil {
		roundTracker.RemoveActiveRound(r.GetRoundID())
	}
	atomic.StoreUint32(scheduleTracker, 19)
	roundId := r.GetRoundID()
	atomic.StoreUint32(scheduleTracker, 20)
	roundInfo := r.BuildRoundInfo()
	atomic.StoreUint32(scheduleTracker, 21)
	// Build the round info and update the network state
	err = state.AddRoundUpdate(roundInfo)
	if err != nil {
		return errors.WithMessagef(err, "Could not issue "+
			"update to kill round %v", r.GetRoundID())
	}
	atomic.StoreUint32(scheduleTracker, 22)

	// Attempt to insert the RoundMetric for the failed round
	metric := &storage.RoundMetric{
		Id:            uint64(roundId),
		PrecompStart:  time.Unix(0, int64(r.BuildRoundInfo().Timestamps[states.PRECOMPUTING])),
		PrecompEnd:    time.Unix(0, int64(r.BuildRoundInfo().Timestamps[states.STANDBY])),
		RealtimeStart: time.Unix(0, int64(r.BuildRoundInfo().Timestamps[states.REALTIME])),
		RealtimeEnd:   time.Unix(0, int64(r.BuildRoundInfo().Timestamps[states.FAILED])),
		BatchSize:     r.BuildRoundInfo().BatchSize,
	}
	atomic.StoreUint32(scheduleTracker, 23)
	err = storage.PermissioningDb.InsertRoundMetric(metric,
		roundInfo.Topology)
	if err != nil {
		jww.WARN.Printf("Could not insert round metric: %+v", err)
		err = nil
	}
	atomic.StoreUint32(scheduleTracker, 24)

	nid, err := id.Unmarshal(roundError.NodeId)
	var idStr string
	if err != nil {
		idStr = "N/A"
	} else {
		idStr = nid.String()
	}
	atomic.StoreUint32(scheduleTracker, 25)

	formattedError := fmt.Sprintf("Round Error from %s: %s", idStr, roundError.Error)
	atomic.StoreUint32(scheduleTracker, 26)

	// Next, attempt to insert the error for the failed round
	err = storage.PermissioningDb.InsertRoundError(roundId, formattedError)
	if err != nil {
		jww.WARN.Printf("Could not insert round error: %+v", err)
		err = nil
	}
	atomic.StoreUint32(scheduleTracker, 27)

	// fix a potential error case where a node crashes after the round is
	// created but before it updates to precomputing and then gets stuck
	topology := r.GetTopology()
	for i := 0; i < topology.Len(); i++ {
		nid := topology.GetNodeAtIndex(i)
		n := state.GetNodeMap().GetNode(nid)
		if n != nil {
			if n.GetActivity() == current.WAITING {
				hasRound, rNode := n.GetCurrentRound()
				if hasRound && rNode.GetRoundID() == r.GetRoundID() {
					n.ClearRound()
					pool.Add(n)
				}
			}
		}
	}
	atomic.StoreUint32(scheduleTracker, 28)

	return err
}
