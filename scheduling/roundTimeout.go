////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package scheduling

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

func waitForRoundTimeout(tracker chan id.Round, state *storage.NetworkState,
	localRound *round.State, timeout time.Duration, isRealtime bool) {
	roundID := localRound.GetRoundID()
	// Allow for round the to be added to the map
	roundTimer := time.NewTimer(timeout)
	select {
	// Wait for the timer to go off
	case <-roundTimer.C:
		// Send the timed out round id to the timeout handler
		jww.INFO.Printf("Round %v[Realtime: %t] has timed out after %s, "+
			"signaling exit", roundID, isRealtime, timeout)
		tracker <- roundID
	// Signals the round has been completed.
	// In this case, we can exit the go-routine
	case <-localRound.GetRoundCompletedChan():
		if isRealtime {
			jww.DEBUG.Printf("[TEST] Removing round %s in waitForRoundTimeout", roundID.String())
			state.GetRoundMap().DeleteRound(roundID)
		}
		return
	}
}
