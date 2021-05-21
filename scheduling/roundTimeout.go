package scheduling

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/xx_network/primitives/id"
	"time"
)


func waitForRoundTimeout(tracker chan id.Round, state *storage.NetworkState,
	localRound *round.State, timeout time.Duration, timoutType string) {
	roundID := localRound.GetRoundID()
	// Allow for round the to be added to the map
	roundTimer := time.NewTimer(timeout)
	select {
	// Wait for the timer to go off
	case <-roundTimer.C:
		// Send the timed out round id to the timeout handler
		jww.INFO.Printf("Round %v has %s timed out after %s, " +
			"signaling exit", roundID, timoutType, timeout)
		tracker <- roundID
	// Signals the round has been completed.
	// In this case, we can exit the go-routine
	case <-localRound.GetRoundCompletedChan():
		if timoutType == "realtime"{
			state.GetRoundMap().DeleteRound(roundID)
		}
		return
	}
}
