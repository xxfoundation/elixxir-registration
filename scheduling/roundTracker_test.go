////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package scheduling

import (
	"gitlab.com/xx_network/primitives/id"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

// Tests the happy path of TestNewRoundTracker().
func TestNewRoundTracker(t *testing.T) {
	// Test values
	expectedActiveRounds := make(map[id.Round]struct{})

	testRT := NewRoundTracker()

	if !reflect.DeepEqual(expectedActiveRounds, testRT.activeRounds) {
		t.Errorf("NewRoundTracker() returned a RoundTracker with unexpecged"+
			"activeRounds.\n\texpected: %v\n\trecieved: %v",
			expectedActiveRounds, testRT)
	}
}

// Tests that AddActiveRound() correctly adds the four random round IDs.
func TestRoundTracker_AddActiveRound(t *testing.T) {
	// Test values
	testRT := NewRoundTracker()
	testRID := []id.Round{id.Round(rand.Uint64()), id.Round(rand.Uint64()),
		id.Round(rand.Uint64()), id.Round(rand.Uint64())}
	expectedActiveRounds := map[id.Round]struct{}{
		testRID[0]: {}, testRID[1]: {}, testRID[2]: {}, testRID[3]: {},
	}

	for _, rid := range testRID {
		testRT.AddActiveRound(rid)
	}

	if !reflect.DeepEqual(expectedActiveRounds, testRT.activeRounds) {
		t.Errorf("AddActiveRound() did not add the round correctly."+
			"\n\texpected: %v\n\trecieved: %v",
			expectedActiveRounds, testRT.activeRounds)
	}
}

// Tests that AddActiveRound() is thread safe.
func TestRoundTracker_AddActiveRound_Thread_Lock(t *testing.T) {
	// Test values
	testRT := NewRoundTracker()
	result := make(chan bool)

	// Lock the thread
	testRT.mux.Lock()
	defer testRT.mux.Unlock()

	go func() {
		testRT.AddActiveRound(0)
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("AddActiveRound() did not correctly lock the thread.")
	case <-time.After(33 * time.Millisecond):
		return
	}
}

// Tests that Len() returns the correct length of map.
func TestRoundTracker_Len(t *testing.T) {
	// Test values
	testRT := NewRoundTracker()
	expectedLen := 12
	for i := 0; i < expectedLen; i++ {
		testRT.AddActiveRound(id.Round(rand.Uint64()))
	}

	testLen := testRT.Len()

	if expectedLen != testLen {
		t.Errorf("Len() returned the incorrect length."+
			"\n\texpected: %d\n\treceived: %d", expectedLen, testLen)
	}
}

// Tests that Len() is thread safe.
func TestRoundTracker_Len_Thread_Lock(t *testing.T) {
	// Test values
	testRT := NewRoundTracker()
	result := make(chan bool)

	// Lock the thread
	testRT.mux.Lock()
	defer testRT.mux.Unlock()

	go func() {
		testRT.Len()
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("Len() did not correctly lock the thread.")
	case <-time.After(33 * time.Millisecond):
		return
	}
}

// Tests that RemoveActiveRound() removes the correct rounds.
func TestRoundTracker_RemoveActiveRound(t *testing.T) {
	// Test values
	testRT := NewRoundTracker()
	expectedActiveRounds := make(map[id.Round]struct{})
	testRIDs := make([]id.Round, 12)
	for i := range testRIDs {
		testRIDs[i] = id.Round(rand.Uint64())
	}

	for i, rid := range testRIDs {
		testRT.AddActiveRound(rid)
		if i%2 == 0 {
			expectedActiveRounds[rid] = struct{}{}
		}
	}

	for i, rid := range testRIDs {
		if i%2 == 1 {
			testRT.RemoveActiveRound(rid)
		}
	}

	if !reflect.DeepEqual(expectedActiveRounds, testRT.activeRounds) {
		t.Errorf("RemoveActiveRound() did not remove the correct elements."+
			"\n\texpected: %v\n\trecieved: %v",
			expectedActiveRounds, testRT.activeRounds)
	}
}

// Tests that RemoveActiveRound() is thread safe.
func TestRoundTracker_RemoveActiveRound_Thread_Lock(t *testing.T) {
	// Test values
	testRT := NewRoundTracker()
	result := make(chan bool)

	// Lock the thread
	testRT.mux.Lock()
	defer testRT.mux.Unlock()

	go func() {
		testRT.RemoveActiveRound(0)
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("RemoveActiveRound() did not correctly lock the thread.")
	case <-time.After(33 * time.Millisecond):
		return
	}
}

func TestRoundTracker_GetActiveRounds(t *testing.T) {
	// Test values
	testRT := NewRoundTracker()
	expectedRIDs := make([]id.Round, 12)
	for i := range expectedRIDs {
		expectedRIDs[i] = id.Round(i)
		testRT.AddActiveRound(expectedRIDs[i])
	}

	testRIDs := testRT.GetActiveRounds()

	if len(compare(expectedRIDs, testRIDs)) != 0 {
		t.Errorf("GetActiveRounds() did not return the correct rounds."+
			"\n\texpected: %v\n\trecieved: %v",
			expectedRIDs, testRIDs)
	}

}

// Tests that GetActiveRounds() is thread safe.
func TestRoundTracker_GetActiveRounds_Thread_Lock(t *testing.T) {
	// Test values
	testRT := NewRoundTracker()
	result := make(chan bool)

	// Lock the thread
	testRT.mux.Lock()
	defer testRT.mux.Unlock()

	go func() {
		_ = testRT.GetActiveRounds()
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("RemoveActiveRound() did not correctly lock the thread.")
	case <-time.After(33 * time.Millisecond):
		return
	}
}

func compare(X, Y []id.Round) []id.Round {
	m := make(map[id.Round]int)

	for _, y := range Y {
		m[y]++
	}

	var ret []id.Round
	for _, x := range X {
		if m[x] > 0 {
			m[x]--
			continue
		}
		ret = append(ret, x)
	}

	return ret
}
