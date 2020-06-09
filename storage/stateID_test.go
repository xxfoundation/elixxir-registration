////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package storage

import (
	"gitlab.com/elixxir/primitives/utils"
	"os"
	"strconv"
	"testing"
	"time"
)

// Tests that loadOrCreateStateID() correctly reads the ID from file and
// constructs the roundID with the correct values.
func TestLoadRoundID(t *testing.T) {
	expectedPath := "testRoundID.txt"
	expectedID := uint64(9843)
	idString := []byte(strconv.FormatUint(expectedID, 10))

	defer func() {
		err := os.RemoveAll(expectedPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}()

	err := utils.WriteFile(expectedPath, idString, utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Failed to write test file: %+v", err)
	}

	testSID, err := loadOrCreateStateID(expectedPath, 0)
	if err != nil {
		t.Errorf("loadOrCreateStateID() produced an unexpected error: %+v", err)
	}

	if expectedID != testSID.id {
		t.Errorf("loadOrCreateStateID() returned a roundID with an incorrect ID."+
			"\n\t expected: %+v\n\treceived: %+v", expectedID, testSID.id)
	}

	if expectedPath != testSID.path {
		t.Errorf("loadOrCreateStateID() returned a roundID with an incorrect path."+
			"\n\t expected: %+v\n\treceived: %+v", expectedPath, testSID.path)
	}
}

// Tests that loadOrCreateStateID() sets the ID to 0 when no path is provided.
func TestLoadRoundID_EmptyPath(t *testing.T) {
	expectedPath := ""
	expectedID := uint64(0)

	testSID, err := loadOrCreateStateID(expectedPath, 0)
	if err != nil {
		t.Errorf("loadOrCreateStateID() produced an unexpected error: %+v", err)
	}

	if expectedID != testSID.id {
		t.Errorf("loadOrCreateStateID() returned a roundID with an incorrect ID."+
			"\n\t expected: %+v\n\treceived: %+v", expectedID, testSID.id)
	}

	if expectedPath != testSID.path {
		t.Errorf("loadOrCreateStateID() returned a roundID with an incorrect path."+
			"\n\t expected: %+v\n\treceived: %+v", expectedPath, testSID.path)
	}
}

// Tests that loadOrCreateStateID() returns an error when the file does not contain a
// uint64.
func TestLoadRoundID_FileContentError(t *testing.T) {
	expectedPath := "testRoundID.txt"
	expectedError := "Could not convert ID to uint: strconv.ParseUint: " +
		"parsing \"test\": invalid syntax"

	defer func() {
		err := os.RemoveAll(expectedPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}()

	err := utils.WriteFile(expectedPath, []byte("test"), utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Failed to write test file: %+v", err)
	}

	_, err = loadOrCreateStateID(expectedPath, 0)
	if err == nil {
		t.Errorf("loadOrCreateStateID() did not produce the expected error."+
			"\n\texpected: %+v\n\treceived: %+v", expectedError, err)
	}
}

// Tests that increment() increments the ID the correct number of times and
// saves the value to file.
func TestRoundID_Increment(t *testing.T) {
	testID := uint64(9843)
	testPath := "testRoundID.txt"
	incrementAmount := uint64(10)
	testSID := stateID{
		id:   testID,
		path: testPath,
	}

	defer func() {
		err := os.RemoveAll(testPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}()

	for i := uint64(0); i < incrementAmount; i++ {
		oldID, err := testSID.increment()
		if err != nil {
			t.Errorf("increment() produced an unexpected error on index %d: "+
				"%+v", i, err)
		}

		// Test that the correct old ID was returned
		if oldID != testID+i {
			t.Errorf("increment() did not return the correct old ID."+
				"\n\texpected: %+v\n\treceived: %+v", testID+i, oldID)
		}

		// Test that the ID in memory was correctly incremented
		if testSID.id != testID+i+1 {
			t.Errorf("increment() did not increment the ID in memory correctly."+
				"\n\texpected: %+v\n\treceived: %+v", testID+i+1, testSID.id)
		}

		// Test that the ID on disk was correctly incremented
		sidBytes, err := utils.ReadFile(testPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		idUint, err := strconv.ParseUint(string(sidBytes), 10, 64)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		if idUint != testID+i+1 {
			t.Errorf("increment() did not increment the ID on disk "+
				"correctly.\n\texpected: %+v\n\treceived: %+v",
				testID+i+1, idUint)
		}
	}
}

// Tests that increment() increments the internal ID the correct number of times
// but skips writing to file when an empty path is provided.
func TestRoundID_Increment_EmptyPath(t *testing.T) {
	testID := uint64(9843)
	testPath := ""
	incrementAmount := uint64(10)
	testSID := stateID{
		id:   testID,
		path: testPath,
	}

	for i := uint64(0); i < incrementAmount; i++ {
		oldID, err := testSID.increment()
		if err != nil {
			t.Errorf("increment() produced an unexpected error on "+
				"index %d: %+v", i, err)
		}

		// Test that the correct old ID was returned
		if oldID != testID+i {
			t.Errorf("increment() did not return the correct old ID."+
				"\n\texpected: %+v\n\treceived: %+v", testID+i, oldID)
		}

		// Test that the ID in memory was correctly incremented
		if testSID.id != testID+i+1 {
			t.Errorf("increment() did not increment the ID in memory correctly."+
				"\n\texpected: %+v\n\treceived: %+v",
				testID+i+1, testSID.id)
		}
	}
}

// Tests that increment() returns an error for an invalid file path and that the ID
// is not updated on error.
func TestRoundID_Increment_FileError(t *testing.T) {
	testID := uint64(9843)
	testPath := "~a/testRoundID.txt"
	testSID := stateID{
		id:   testID,
		path: testPath,
	}

	defer func() {
		err := os.RemoveAll(testPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}()

	_, err := testSID.increment()
	if err == nil {
		t.Errorf("increment() did not produce an error on an invalid path.")
	}

	if testSID.id != testID {
		t.Errorf("increment() unexpectedly incremented the ID on error."+
			"\n\texpected: %+v\n\treceived: %+v", testID, testSID.id)
	}
}

// Tests that increment() blocks when the thread is locked.
func TestRoundID_Increment_Lock(t *testing.T) {
	expectedID := uint64(9843)
	testSID := stateID{
		id:   expectedID,
		path: "testRoundID.txt",
	}

	result := make(chan bool)

	testSID.Lock()

	go func() {
		_, _ = testSID.increment()
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("increment() did not correctly lock the thread.")
	case <-time.After(time.Second):
		return
	}
}

// Tests that get() returns the correct value.
func TestRoundID_Get(t *testing.T) {
	expectedID := uint64(9843)
	testSID := stateID{
		id:   expectedID,
		path: "testRoundID.txt",
	}

	testID := testSID.get()

	if expectedID != testID {
		t.Errorf("get() returned an incorrect ID."+
			"\n\texpected: %+v\n\treceived: %+v", expectedID, testID)
	}
}

// Tests that get() blocks when the thread is locked.
func TestRoundID_Get_Lock(t *testing.T) {
	expectedID := uint64(9843)
	testSID := stateID{
		id:   expectedID,
		path: "testRoundID.txt",
	}

	result := make(chan bool)

	testSID.Lock()

	go func() {
		_ = testSID.get()
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("get() did not correctly lock the thread.")
	case <-time.After(time.Second):
		return
	}
}

// Tests that calling loadOrCreateStateID() multiple times on a previously
// incremented ID files results in the correct ID. This simulates what the ID
// file will do in integration.
func TestRoundID_IntegrationSim(t *testing.T) {
	testID := uint64(9843)
	testPath := "testRoundID.txt"
	idString := []byte(strconv.FormatUint(testID, 10))
	incrementAmount := uint64(10)
	idTracker := testID

	defer func() {
		err := os.RemoveAll(testPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}()

	err := utils.WriteFile(testPath, idString, utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Failed to write test file: %+v", err)
	}

	for i := 0; i < 5; i++ {
		testSID, err2 := loadOrCreateStateID(testPath, 0)
		if err2 != nil {
			t.Errorf("loadOrCreateStateID() produced an unexpected error at "+
				"index %d: %+v", i, err2)
		}

		if testSID.id != idTracker {
			t.Errorf("loadOrCreateStateID() produced a state ID with an "+
				"incorrect ID at index %d.\n\texpected: %+v\n\treceived: %+v",
				i, idTracker, testSID.id)
		}

		for j := uint64(0); j < incrementAmount; j++ {
			_, err = testSID.increment()
			if err != nil {
				t.Errorf("increment() produced an unexpected error on index "+
					"%d: %+v", j, err)
			}
		}

		idTracker += incrementAmount

		if testSID.id != idTracker {
			t.Errorf("increment() did not increment the id correctly at index "+
				"%d.\n\texpected: %+v\n\treceived: %+v",
				i, idTracker, testSID.id)
		}
	}
}

// Tests that calling loadOrCreateStateID() multiple times on a previously
// incremented ID files results in the correct ID. This simulates what the ID
// file will do in integration.
func TestRoundID_IntegrationSim_NoFile(t *testing.T) {
	testPath := "testRoundID.txt"
	incrementAmount := uint64(10)
	idTracker := uint64(0)

	defer func() {
		err := os.RemoveAll(testPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}()

	for i := 0; i < 5; i++ {
		testSID, err2 := loadOrCreateStateID(testPath, 0)
		if err2 != nil {
			t.Errorf("loadOrCreateStateID() produced an unexpected error at "+
				"index %d: %+v", i, err2)
		}

		if testSID.id != idTracker {
			t.Errorf("loadOrCreateStateID() produced a state ID with an "+
				"incorrect ID at index %d.\n\texpected: %+v\n\treceived: %+v",
				i, idTracker, testSID.id)
		}

		for j := uint64(0); j < incrementAmount; j++ {
			_, err := testSID.increment()
			if err != nil {
				t.Errorf("increment() produced an unexpected error on index "+
					"%d: %+v", j, err)
			}
		}

		idTracker += incrementAmount

		if testSID.id != idTracker {
			t.Errorf("increment() did not increment the id correctly at index "+
				"%d.\n\texpected: %+v\n\treceived: %+v",
				i, idTracker, testSID.id)
		}
	}
}
