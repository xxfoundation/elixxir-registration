////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

func TestNewRoundID(t *testing.T) {
	startVal := 1
	startRound := id.Round(startVal)
	receivedRound := NewRoundID(startRound)

	// Test get
	if receivedRound.Get() != startRound {
		t.Errorf("New round does not have expected starting value. "+
			"\n\tReceived value: %d"+
			"\n\tExpected value: %d", receivedRound.Get(), startRound)
	}

	oldRound := receivedRound.Next()

	// Test next returns old value
	if oldRound != startRound {
		t.Errorf("Next did not return expected old value."+
			"\n\tReceived value: %d"+
			"\n\tExpected value: %d", oldRound, startRound)
	}

	newRound := receivedRound.Get()
	// Test that next incremented value
	if newRound != id.Round(startVal+1) {
		t.Errorf("Next did not increment value."+
			"\n\tReceived value: %d"+
			"\n\tExpected value: %d", newRound, id.Round(startVal+1))
	}
}
