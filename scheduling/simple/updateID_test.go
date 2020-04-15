////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

func TestNewUpdateID(t *testing.T) {
	startVal := uint64(1)

	receivedID := NewUpdateID(startVal)

	// Test get
	if receivedID.Get() != startVal {
		t.Errorf("New updateID does not have expected starting value. "+
			"\n\tReceived value: %d"+
			"\n\tExpected value: %d", receivedID.Get(), startVal)
	}

	oldId := receivedID.Next()

	// Test next returns old value
	if oldId != startVal {
		t.Errorf("Next did not return expected old value."+
			"\n\tReceived value: %d"+
			"\n\tExpected value: %d", oldId, startVal)
	}

	newID := receivedID.Get()
	// Test that next incremented value
	if newID != startVal+1 {
		t.Errorf("Next did not increment value."+
			"\n\tReceived value: %d"+
			"\n\tExpected value: %d", newID, id.Round(startVal+1))
	}

}
