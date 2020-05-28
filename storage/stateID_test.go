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

	testRID, err := loadOrCreateStateID(expectedPath)
	if err != nil {
		t.Errorf("loadOrCreateStateID() produced an unexpected error: %+v", err)
	}

	if expectedID != testRID.id {
		t.Errorf("loadOrCreateStateID() returned a roundID with an incorrect ID."+
			"\n\t expected: %+v\n\treceived: %+v", expectedID, testRID.id)
	}

	if expectedPath != testRID.path {
		t.Errorf("loadOrCreateStateID() returned a roundID with an incorrect path."+
			"\n\t expected: %+v\n\treceived: %+v", expectedPath, testRID.path)
	}
}

// Tests that loadOrCreateStateID() sets the ID to 0 when no path is provided.
func TestLoadRoundID_EmptyPath(t *testing.T) {
	expectedPath := ""
	expectedID := uint64(0)

	testRID, err := loadOrCreateStateID(expectedPath)
	if err != nil {
		t.Errorf("loadOrCreateStateID() produced an unexpected error: %+v", err)
	}

	if expectedID != testRID.id {
		t.Errorf("loadOrCreateStateID() returned a roundID with an incorrect ID."+
			"\n\t expected: %+v\n\treceived: %+v", expectedID, testRID.id)
	}

	if expectedPath != testRID.path {
		t.Errorf("loadOrCreateStateID() returned a roundID with an incorrect path."+
			"\n\t expected: %+v\n\treceived: %+v", expectedPath, testRID.path)
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

	_, err = loadOrCreateStateID(expectedPath)
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
	testRID := stateID{
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
		oldID, err := testRID.increment()
		if err != nil {
			t.Errorf("increment() produced an unexpected error on  index %d: "+
				"%+v", i, err)
		}

		// Test that the correct old ID was returned
		if oldID != testID+i {
			t.Errorf("increment() did not return the correct old ID."+
				"\n\texpected: %+v\n\treceived: %+v", testID+i, oldID)
		}

		// Test that the ID in memory was correctly incremented
		if testRID.id != testID+i+1 {
			t.Errorf("increment() did not increment the ID in memory correctly."+
				"\n\texpected: %+v\n\treceived: %+v", testID+i+1, testRID.id)
		}

		// Test that the ID on disk was correctly incremented
		ridBytes, err := utils.ReadFile(testPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
		idUint, err := strconv.ParseUint(string(ridBytes), 10, 64)
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
	testRID := stateID{
		id:   testID,
		path: testPath,
	}

	for i := uint64(0); i < incrementAmount; i++ {
		oldID, err := testRID.increment()
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
		if testRID.id != testID+i+1 {
			t.Errorf("increment() did not increment the ID in memory correctly."+
				"\n\texpected: %+v\n\treceived: %+v",
				testID+i+1, testRID.id)
		}
	}
}

// Tests that increment() returns an error for an invalid file path and that the ID
// is not updated on error.
func TestRoundID_Increment_FileError(t *testing.T) {
	testID := uint64(9843)
	testPath := "~a/testRoundID.txt"
	testRID := stateID{
		id:   testID,
		path: testPath,
	}

	defer func() {
		err := os.RemoveAll(testPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}()

	_, err := testRID.increment()
	if err == nil {
		t.Errorf("increment() did not produce an error on an invalid path.")
	}

	if testRID.id != testID {
		t.Errorf("increment() unexpectedly incremented the ID on error."+
			"\n\texpected: %+v\n\treceived: %+v", testID, testRID.id)
	}
}

// Tests that increment() blocks when the thread is locked.
func TestRoundID_Increment_Lock(t *testing.T) {
	expectedID := uint64(9843)
	testRID := stateID{
		id:   expectedID,
		path: "testRoundID.txt",
	}

	result := make(chan bool)

	testRID.Lock()

	go func() {
		_, _ = testRID.increment()
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
	testRID := stateID{
		id:   expectedID,
		path: "testRoundID.txt",
	}

	testID := testRID.get()

	if expectedID != testID {
		t.Errorf("get() returned an incorrect ID."+
			"\n\texpected: %+v\n\treceived: %+v", expectedID, testID)
	}
}

// Tests that get() blocks when the thread is locked.
func TestRoundID_Get_Lock(t *testing.T) {
	expectedID := uint64(9843)
	testRID := stateID{
		id:   expectedID,
		path: "testRoundID.txt",
	}

	result := make(chan bool)

	testRID.Lock()

	go func() {
		_ = testRID.get()
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("get() did not correctly lock the thread.")
	case <-time.After(time.Second):
		return
	}
}
