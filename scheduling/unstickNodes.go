////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Contains a fix for a bug where nodes get stuck in waiting
package scheduling

import (
	"git.xx.network/elixxir/primitives/current"
	"git.xx.network/elixxir/registration/storage"
	"git.xx.network/elixxir/registration/storage/node"
	"time"
)

// Detect nodes that are stuck on waiting with no current round running
// Then, add them back to the active pool so they can run rounds again
func unstickNodes(state *storage.NetworkState, pool *waitingPool, roundTimeout time.Duration) {
	// Check states of all nodes
	nodeStates := state.GetNodeMap().GetNodeStates()
	for i := range nodeStates {
		thisNode := nodeStates[i]
		_, thisRound := nodeStates[i].GetCurrentRound()
		// add the node back to the waiting pool if certain conditions are met
		if thisRound != nil && time.Since(thisRound.GetLastUpdate()) > 2*roundTimeout && thisNode.GetStatus() == node.Active && thisNode.GetActivity() == current.WAITING {
			thisNode.ClearRound()
			pool.Add(thisNode)
		}
	}
}

// Runner that unsticks nodes periodically
// Exported methods should have only exported types for params! Right?
func UnstickNodes(state *storage.NetworkState, pool *waitingPool, roundTimeout time.Duration, quitChan chan struct{}) {
	unstickNodeTicker := time.NewTicker(2 * roundTimeout)
	for cont := true; cont; {
		select {
		case <-unstickNodeTicker.C:
			unstickNodes(state, pool, roundTimeout)
		case <-quitChan:
			cont = false
		}
	}
}
