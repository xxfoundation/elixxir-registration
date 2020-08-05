////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"crypto/rand"
	"github.com/golang-collections/collections/set"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Test that generateDisabledNodes() correctly generates a disabledNodes
// object from a file.
func TestGenerateDisabledNodes(t *testing.T) {
	// Get test data
	testData, stateMap, expectedStateSet := generateIdLists(3, t)
	testData = "\n \n\n" + testData + "\n  "
	testPath := "testDisabledNodesList.txt"

	// Delete the test file at the end
	defer func() {
		err := os.RemoveAll(testPath)
		if err != nil {
			t.Fatalf("Error deleting test file %#v:\n%v", testPath, err)
		}
	}()

	// Create test file
	err := utils.WriteFile(testPath, []byte(testData), utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Error while creating test file: %v", err)
	}

	dnl, err := generateDisabledNodes(testPath, 33*time.Millisecond, &NetworkState{nodes: stateMap})
	if err != nil {
		t.Errorf("generateDisabledNodes() produced an unexpected error: %v", err)
	}

	if !reflect.DeepEqual(dnl.nodes, expectedStateSet) {
		t.Errorf("getDisabledNodes() did not return the correct values "+
			"from the disabled Node list.\n\texpected: %v\n\treceived: %v",
			expectedStateSet, dnl.nodes)
	}
}

// Tests that generateDisabledNodes() returns an error if the file cannot be
// found.
func TestGenerateDisabledNodes_FileError(t *testing.T) {
	testPath := "testDisabledNodesList.txt"
	expectedError := "Skipping polling of disabled node ID list file; error " +
		"while accessing file: open " + testPath + ": The system cannot find " +
		"the file specified."

	dnl, err := generateDisabledNodes(testPath, 33*time.Millisecond, nil)
	if err == nil {
		t.Errorf("generateDisabledNodes() did not produce an error when "+
			"expected.\n\texpected: %v\n\treceived: %v", expectedError, err)
	}

	if dnl != nil {
		t.Errorf("generateDisabledNodes() did not return an empty list on error."+
			"\n\texpected: %v\n\treceived: %v",
			nil, dnl)
	}
}

// Tests that generateDisabledNodes() generates a disabledNodes when ID
// parsing errors occur.
func TestGenerateDisabledNodes_IdWarning(t *testing.T) {
	// Get test data
	testData, stateMap, expectedStateSet := generateIdLists(3, t)
	testData = "\na\nNoKjAhvURKnrwdLIvBe8AF9gTEV6qPRtgcXEKCRh620=\n" +
		"TRlATuYybZfN2JznUcrAws5DpfesA2tzc6b/rp3jqv8A\n" + testData + "test"
	testPath := "testDisabledNodesList.txt"

	// Delete the test file at the end
	defer func() {
		err := os.RemoveAll(testPath)
		if err != nil {
			t.Fatalf("Error deleting test file %#v:\n%v", testPath, err)
		}
	}()

	// Create test file
	err := utils.WriteFile(testPath, []byte(testData), utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Error while creating test file: %v", err)
	}

	dnl, err := generateDisabledNodes(testPath, 33*time.Millisecond, &NetworkState{nodes: stateMap})
	if err != nil {
		t.Errorf("generateDisabledNodes() produced an unexpected error: %v", err)
	}

	if !reflect.DeepEqual(dnl.nodes, expectedStateSet) {
		t.Errorf("getDisabledNodes() did not return the correct values "+
			"from the disabled Node list.\n\texpected: %v\n\treceived: %v",
			expectedStateSet, dnl.nodes)
	}
}

// Tests getDisabledNodesSet() happy path.
func TestGetDisabledNodesSet(t *testing.T) {
	// Get test data
	testData, testStateMap, expectedStateSet := generateIdLists(3, t)

	testStateSet, err := getDisabledNodesSet(testData, testStateMap)
	if err != nil {
		t.Errorf("getDisabledNodesSet() produced an unexpected error: %v", err)
	}

	if !testStateSet.SubsetOf(expectedStateSet) {
		t.Errorf("getDisabledNodesSet() produced an incorrect set."+
			"\n\texpected: %v\n\treceived: %v",
			expectedStateSet, testStateSet)
	}
}

// Tests that getDisabledNodesSet() produces an error when provided three types
// of invalid IDs: (1) string that cannot be base64 decoded, (2) a byte slice
// that is too short, and (3) an ID that is not in the state map. Also tests
// that a nil set is returned when no IDs were found.
func TestGetDisabledNodesSet_IdError(t *testing.T) {
	// Test values
	testData := "\na\nNoKjAhvURKnrwdLIvBe8AF9gTEV6qPRtgcXEKCRh620=\n" +
		"TRlATuYybZfN2JznUcrAws5DpfesA2tzc6b/rp3jqv8A\n"

	testIdList, err := getDisabledNodesSet(testData, &node.StateMap{})
	if err == nil {
		t.Errorf("getDisabledNodesSet() did not produce an error when expected.")
	}

	if err != nil && !strings.Contains(err.Error(), "base64") {
		t.Errorf("getDisabledNodesSet() did not produce a base64 decode error."+
			"\n\treceived: %v", err)
	}

	if err != nil && !strings.Contains(err.Error(), "unmarshal") {
		t.Errorf("getDisabledNodesSet() did not produce an unmarshal error."+
			"\n\treceived: %v", err)
	}

	if err != nil && !strings.Contains(err.Error(), "state map") {
		t.Errorf("getDisabledNodesSet() did not produce a state map error."+
			"\n\treceived: %v", err)
	}

	if testIdList != nil {
		t.Errorf("getDisabledNodesSet() produced a non-nil ID list on error."+
			"\n\texpected: %v\n\treceived: %v", nil, testIdList)
	}
}

// Tests that updateDisabledNodesList copies the values from the new list
// instead of just the reference.
func TestDisabledNodes_UpdateDisabledNodesList(t *testing.T) {
	// Get test data
	_, _, initialStateSet := generateIdLists(3, t)
	_, _, testStateSet := generateIdLists(3, t)

	dnl := disabledNodes{nodes: initialStateSet}

	dnl.updateDisabledNodes(testStateSet)

	if &testStateSet == &dnl.nodes {
		t.Errorf("updateDisabledNodesList() copied the pointer instead of " +
			"the values of the new ID list.")
	}

	if !reflect.DeepEqual(testStateSet, dnl.nodes) {
		t.Errorf("updateDisabledNodesList() did not copy the correct values "+
			"from the new ID list.\n\texpected: %v\n\treceived: %v",
			testStateSet, dnl.nodes)
	}
}

// Tests that updateDisabledNodesList correctly locks the thread.
func TestDisabledNodes_UpdateDisabledNodesList_Lock(t *testing.T) {
	// Get test data
	_, _, initialStateSet := generateIdLists(3, t)
	_, _, testStateSet := generateIdLists(3, t)

	dnl := disabledNodes{nodes: initialStateSet}

	result := make(chan bool)

	dnl.RLock()

	go func() {
		dnl.updateDisabledNodes(testStateSet)
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("updateDisabledNodesList() did not correctly lock the thread.")
	case <-time.After(500 * time.Millisecond):
		return
	}
}

// Tests that getDisabledNodes gets a copy of the disabled Node ID list.
func TestDisabledNodes_GetDisabledNodesList(t *testing.T) {
	// Get test data
	_, _, expectedStateSet := generateIdLists(3, t)
	dnl := disabledNodes{nodes: expectedStateSet}

	testIdList := dnl.getDisabledNodes()

	if &testIdList == &expectedStateSet {
		t.Errorf("getDisabledNodes() copied the pointer instead of the " +
			"values of disabled Node list.")
	}

	if !reflect.DeepEqual(testIdList, expectedStateSet) {
		t.Errorf("getDisabledNodes() did not return the correct values "+
			"from the disabled Node list.\n\texpected: %v\n\treceived: %v",
			expectedStateSet, testIdList)
	}
}

// Tests that getDisabledNodes correctly locks the thread.
func TestDisabledNodes_GetDisabledNodesList_Lock(t *testing.T) {
	// Get test data
	_, _, expectedStateSet := generateIdLists(3, t)
	dnl := disabledNodes{nodes: expectedStateSet}

	result := make(chan bool)

	dnl.Lock()

	go func() {
		_ = dnl.getDisabledNodes()
		result <- true
	}()

	select {
	case <-result:
		t.Errorf("getDisabledNodes() did not correctly lock the thread.")
	case <-time.After(500 * time.Millisecond):
		return
	}
}

// Tests that pollDisabledNodes() correctly initialises the list and process
// which checks for updates to the list file. The file is updated twice. The
// first time with three valid IDs with extra whitespace on both ends. The
// second has an extra invalid ID that is expected to be skipped.
func TestDisabledNodes_PollDisabledNodes(t *testing.T) {
	// Get test data
	testData1, stateMap1, initialStateSet := generateIdLists(3, t)
	testData1 = "\n \n\n" + testData1 + "\n  "
	testData2 := "\na\nNoKjAhvURKnrwdLIvBe8AF9gTEV6qPRtgcXEKCRh620=\n" +
		"TRlATuYybZfN2JznUcrAws5DpfesA2tzc6b/rp3jqv8A\n" + testData1 + "test"
	dnl := disabledNodes{
		nodes:    nil,
		path:     "testDisabledNodesList.txt",
		interval: 33 * time.Millisecond,
	}

	// Delete the test file at the end
	defer func() {
		err := os.RemoveAll(dnl.path)
		if err != nil {
			t.Fatalf("Error deleting test file %#v:\n%v", dnl.path, err)
		}
	}()

	// Create test file
	err := utils.WriteFile(dnl.path, []byte(testData1), utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Error while creating test file: %v", err)
	}

	go dnl.pollDisabledNodes(&NetworkState{nodes: stateMap1}, nil)

	time.Sleep(100 * time.Millisecond)

	// Get initial values
	testList := dnl.getDisabledNodes()

	if !reflect.DeepEqual(testList, initialStateSet) {
		t.Errorf("getDisabledNodes() did not return the correct values "+
			"from the disabled Node list.\n\texpected: %v\n\treceived: %v",
			initialStateSet, testList)
	}

	// Write new data
	err = utils.WriteFile(dnl.path, []byte(testData2), utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Error while creating test file: %v", err)
	}

	// Wait for the thread to update
	time.Sleep(100 * time.Millisecond)

	testList = dnl.getDisabledNodes()

	if !reflect.DeepEqual(testList, initialStateSet) {
		t.Errorf("getDisabledNodes() did not return the correct values "+
			"from the disabled Node list.\n\texpected: %v\n\treceived: %v",
			initialStateSet, testList)
	}
}

// Tests that pollDisabledNodes() returns an error if the file cannot be found.
func TestDisabledNodes_PollDisabledNodes_FileError(t *testing.T) {
	dnl := disabledNodes{
		nodes:    nil,
		path:     "testDisabledNodesList.txt",
		interval: 33 * time.Millisecond,
	}

	go dnl.pollDisabledNodes(&NetworkState{}, nil)

	time.Sleep(100 * time.Millisecond)

	if dnl.nodes != nil {
		t.Errorf("pollDisabledNodes() did not return an empty list on error."+
			"\n\texpected: %v\n\treceived: %v",
			nil, dnl.nodes)
	}

}

// Tests that pollDisabledNodes() correctly stops looping when triggering the
// quit channel.
func TestDisabledNodes_PollDisabledNodes_QuitChan(t *testing.T) {
	// Get test data
	dnl := disabledNodes{
		nodes:    nil,
		path:     "testDisabledNodesList.txt",
		interval: 33 * time.Millisecond,
	}

	result := make(chan bool)
	quit := make(chan struct{})

	go func() {
		dnl.pollDisabledNodes(&NetworkState{}, quit)
		result <- true
	}()

	quit <- struct{}{}

	select {
	case <-result:
		return
	case <-time.After(500 * time.Millisecond):
		t.Errorf("pollDisabledNodes() did not correctly stop when kill command sent.")
	}
}

func generateIdLists(num int, x interface{}) (string, *node.StateMap, *set.Set) {
	// Generate array of IDs
	var idList, idListL []*id.ID
	randID := make([]byte, 33)
	for i := 0; i < num; i++ {
		_, _ = rand.Read(randID)
		idList = append(idList, id.NewIdFromBytes(randID, x))
		idListL = append(idListL, id.NewIdFromBytes(randID, x))
		_, _ = rand.Read(randID)
		idListL = append(idListL, id.NewIdFromBytes(randID, x))
	}

	// Generate test ID file contents
	var fileData string
	for _, nID := range idList {
		fileData += nID.String() + "\n"
	}

	// Generate test StateMap
	stateMap := node.NewStateMap()
	for _, nID := range idListL {
		_ = stateMap.AddNode(nID, "", "", "", 0)
	}

	// Generate test state Set
	stateSet := set.New()
	for _, nID := range idList {
		stateSet.Insert(stateMap.GetNode(nID))
	}

	return fileData, stateMap, stateSet
}
