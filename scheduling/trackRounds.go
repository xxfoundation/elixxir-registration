package scheduling

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"sync/atomic"
	"time"
)

// Tracks rounds, periodically outputs how many teams are in various rounds
func trackRounds(params Params, state *storage.NetworkState, pool *waitingPool,
	roundTracker *RoundTracker, schedulerIteration *uint32) {
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

		numIterations := atomic.SwapUint32(schedulerIteration, 0)

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

			if now.After(lastPoll) {
				pollDelta = now.Sub(lastPoll)
			}

			if now.After(lastUpdate) {
				updateDelta = now.Sub(lastUpdate)
			}

			if pollDelta > timeToInactive {
				s := fmt.Sprintf("\tNode %s (AppID: %v, Activity: %s) has not polled for %s", nodeState.GetID(), nodeState.GetAppID(), nodeState.GetActivity(), pollDelta)
				if hasround, r := nodeState.GetCurrentRound(); hasround {
					s = fmt.Sprintf("%s, has round %v", s, r.GetRoundID())
				}
				noPoll = append(noPoll, s)
			} else if updateDelta > timeToInactive {
				s := fmt.Sprintf("\tNode %s (AppID: %v) stuck in %s for %s (last poll: %s)", nodeState.GetID(), nodeState.GetAppID(), nodeState.GetActivity(), updateDelta, pollDelta)
				if hasround, r := nodeState.GetCurrentRound(); hasround {
					s = fmt.Sprintf("%s, has round %v", s, r.GetRoundID())
				}
				notUpdating = append(notUpdating, s)
			} else {
				s := fmt.Sprintf("\tNode %s (AppID: %v) operating correctly in %s for %s (last poll: %s)", nodeState.GetID(), nodeState.GetAppID(), nodeState.GetActivity(), updateDelta, pollDelta)
				if hasround, r := nodeState.GetCurrentRound(); hasround {
					s = fmt.Sprintf("%s, has round %v", s, r.GetRoundID())
				}
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
		jww.INFO.Printf("")
		jww.INFO.Printf("Scheduler interations since last update: %v", numIterations)
		jww.INFO.Printf("")
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
		jww.INFO.Printf("Normally operating nodes: %v", len(nodeStates)-len(noPoll)-len(notUpdating))
		jww.INFO.Printf("")

		if len(goodNode) > 0 {
			jww.INFO.Printf("Nodes operating as expected")
			for _, s := range goodNode {
				jww.INFO.Print(s)
			}
			jww.INFO.Printf("")
		}
		if len(noPoll) > 0 {
			jww.INFO.Printf("Nodes with no polls in: %s", timeToInactive)
			for _, s := range noPoll {
				jww.INFO.Print(s)
			}
			jww.INFO.Printf("")
		}

		if len(notUpdating) > 0 {
			jww.INFO.Printf("Nodes with no state updates in: %s", timeToInactive)
			for _, s := range notUpdating {
				jww.INFO.Print(s)
			}
			jww.INFO.Printf("")
		}

		if len(noContact) > 0 {
			jww.INFO.Printf("Nodes which are not included due to no contact error")
			for _, s := range noContact {
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
