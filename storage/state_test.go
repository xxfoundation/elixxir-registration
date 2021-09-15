////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/comms/testkeys"
	"gitlab.com/elixxir/comms/testutils"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/region"
	"gitlab.com/xx_network/primitives/utils"
	mrand "math/rand"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Tests that NewState() creates a new NetworkState with all the correctly
// initialised fields. None of the error paths of NewState() can be tested.
func TestNewState(t *testing.T) {
	// Set up expected values
	expectedRounds := round.NewStateMap()
	expectedRoundUpdates := dataStructures.NewUpdates()
	expectedNodes := node.NewStateMap()
	expectedFullNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		t.Fatalf("Failed to generate new NDF:\n%v", err)
	}
	expectedPartialNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		t.Fatalf("Failed to generate new NDF:\n%v", err)
	}

	PermissioningDb, _, err = NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf(err.Error())
	}

	// Generate private RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key:\n%v", err)
	}

	// Generate new NetworkState
	state, err := NewState(privateKey, 8, "", region.GetCountryBins())
	if err != nil {
		t.Errorf("NewState() produced an unexpected error:\n%v", err)
	}

	// Test fields of NetworkState
	if !reflect.DeepEqual(state.rsaPrivateKey, privateKey) {
		t.Errorf("NewState() produced a NetworkState with the wrong privateKey."+
			"\n\texpected: %v\n\treceived: %v", privateKey, &state.rsaPrivateKey)
	}

	if !reflect.DeepEqual(state.rounds, expectedRounds) {
		t.Errorf("NewState() produced a NetworkState with the wrong rounds."+
			"\n\texpected: %v\n\treceived: %v", expectedRounds, state.rounds)
	}

	// Can't check roundUpdates directly because it contains pointers. Instead,
	// check that it is nto nil and the
	if state.roundUpdates == nil {
		t.Errorf("NewState() produced a NetworkState with a nil roundUpdates."+
			"\n\texpected: %#v\n\treceived: %#v", expectedRoundUpdates, state.roundUpdates)
	}

	lastUpdateID := state.roundUpdates.GetLastUpdateID()
	if lastUpdateID != 0 {
		t.Errorf("roundUpdates has the wrong lastUpdateID"+
			"\n\texpected: %#v\n\treceived: %#v", 0, lastUpdateID)
	}

	if !reflect.DeepEqual(state.nodes, expectedNodes) {
		t.Errorf("NewState() produced a NetworkState with the wrong nodes."+
			"\n\texpected: %v\n\treceived: %v", expectedNodes, state.nodes)
	}

	if !reflect.DeepEqual(state.fullNdf, expectedFullNdf) {
		t.Errorf("NewState() produced a NetworkState with the wrong fullNdf."+
			"\n\texpected: %v\n\treceived: %v", expectedFullNdf, state.fullNdf)
	}

	if !reflect.DeepEqual(state.partialNdf, expectedPartialNdf) {
		t.Errorf("NewState() produced a NetworkState with the wrong partialNdf."+
			"\n\texpected: %v\n\treceived: %v", expectedPartialNdf, state.partialNdf)
	}
}

// Tests that GetFullNdf() returns the correct NDF for a newly created
// NetworkState.
func TestNetworkState_GetFullNdf(t *testing.T) {
	// Set up expected values
	expectedFullNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		t.Fatalf("Failed to generate new NDF:\n%v", err)
	}

	// Generate new and NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Call GetFullNdf()
	fullNdf := state.GetFullNdf()

	if !reflect.DeepEqual(fullNdf, expectedFullNdf) {
		t.Errorf("GetFullNdf() returned the wrong NDF."+
			"\n\texpected: %v\n\treceived: %v", expectedFullNdf, fullNdf)
	}
}

// Tests that GetPartialNdf() returns the correct NDF for a newly created
// NetworkState.
func TestNetworkState_GetPartialNdf(t *testing.T) {
	// Set up expected values
	expectedPartialNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		t.Fatalf("Failed to generate new NDF:\n%v", err)
	}

	// Generate new NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	partialNdf := state.GetPartialNdf()

	if !reflect.DeepEqual(partialNdf, expectedPartialNdf) {
		t.Errorf("GetPartialNdf() returned the wrong NDF."+
			"\n\texpected: %v\n\treceived: %v", expectedPartialNdf, partialNdf)
	}
}

// Smoke test of GetUpdates() by adding rounds and then calling GetUpdates().
func TestNetworkState_GetUpdates(t *testing.T) {
	// Generate new NetworkState
	state, privKey, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Update the round three times and build expected values array
	var expectedRoundInfo []*pb.RoundInfo
	for i := 0; i < 3; i++ {
		roundInfo := &pb.RoundInfo{
			ID:       0,
			UpdateID: uint64(3 + i),
		}
		err = testutils.SignRoundInfoRsa(roundInfo, t)
		if err != nil {
			t.Errorf("Failed to sign round info: %v", err)
			t.FailNow()
		}
		rnd := dataStructures.NewVerifiedRound(roundInfo, privKey.GetPublic())
		err = state.roundUpdates.AddRound(rnd)
		if err != nil {
			t.Errorf("AddRound() produced an unexpected error:\n%+v", err)
			t.FailNow()
		}

		expectedRoundInfo = append(expectedRoundInfo, roundInfo)
	}

	// Test GetUpdates()
	roundInfo, err := state.GetUpdates(2)
	if err != nil {
		t.Errorf("GetUpdates() produced an unexpected error:\n%v", err)
	}

	if !reflect.DeepEqual(roundInfo, expectedRoundInfo) {
		t.Errorf("GetUpdates() returned an incorrect RoundInfo slice."+
			"\n\texpected: %+v\n\treceived: %+v", expectedRoundInfo, roundInfo)
	}
}

// Tests that AddRoundUpdate() by adding a round, checking that it is correct
// and verifying the signature.
func TestNetworkState_AddRoundUpdate(t *testing.T) {
	// Expected Values
	testUpdateID := uint64(1)
	testRoundInfo := &pb.RoundInfo{
		ID:       0,
		UpdateID: 5,
	}
	expectedRoundInfo := *testRoundInfo
	expectedRoundInfo.UpdateID = testUpdateID

	// Generate new private RSA key and NetworkState
	state, privateKey, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Call AddRoundUpdate()
	err = state.AddRoundUpdate(testRoundInfo)
	if err != nil {
		t.Errorf("AddRoundUpdate() unexpectedly produced an error:\n%+v",
			err)
	}
	time.Sleep(100 * time.Millisecond)

	// Test if the round was added
	roundInfoArr, err := state.GetUpdates(0)
	if err != nil {
		t.Fatalf("GetUpdates() produced an unexpected error:\n%+v", err)
	}

	roundInfo := roundInfoArr[0]

	// Make signatures equal because they will not be tested for equality
	expectedRoundInfo.Signature = roundInfo.Signature

	// Check that the round info returned is correct.
	if roundInfo.State != expectedRoundInfo.State && roundInfo.ID != expectedRoundInfo.ID &&
		reflect.DeepEqual(roundInfo.Topology, expectedRoundInfo.Topology) &&
		roundInfo.BatchSize != expectedRoundInfo.BatchSize &&
		reflect.DeepEqual(roundInfo.Errors, expectedRoundInfo.Errors) &&
		reflect.DeepEqual(roundInfo.Timestamps, expectedRoundInfo.Timestamps) &&
		roundInfo.UpdateID != expectedRoundInfo.UpdateID {

		t.Errorf("AddRoundUpdate() added incorrect roundInfo."+
			"\n\texpected: %#v\n\treceived: %#v", expectedRoundInfo, *roundInfo)
	}

	// Verify signature
	err = signature.VerifyRsa(roundInfo, privateKey.GetPublic())
	if err != nil {
		t.Fatalf("Failed to verify RoundInfo signature:\n%+v", err)
	}
}

// Tests that UpdateNdf() updates fullNdf and partialNdf correctly.
func TestNetworkState_UpdateNdf(t *testing.T) {
	// Expected values
	testNDF := &ndf.NetworkDefinition{}

	// Generate new NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Update NDF
	err = state.UpdateNdf(testNDF)
	if err != nil {
		t.Errorf("UpdateNdf() unexpectedly produced an error:\n%+v", err)
	}

	if !reflect.DeepEqual(*state.fullNdf.Get(), *testNDF) {
		t.Errorf("UpdateNdf() saved the wrong NDF fullNdf."+
			"\n\texpected: %#v\n\treceived: %#v", *testNDF, *state.fullNdf.Get())
	}

	if !reflect.DeepEqual(*state.partialNdf.Get(), *testNDF) {
		t.Errorf("UpdateNdf() saved the wrong NDF partialNdf."+
			"\n\texpected: %#v\n\treceived: %#v", *testNDF, *state.partialNdf.Get())
	}
}

// Tests that UpdateNdf() generates an error when injected with invalid private
// key.
func TestNetworkState_UpdateNdf_SignError(t *testing.T) {
	// Expected values
	testNDF := &ndf.NetworkDefinition{}
	expectedErr := "Unable to sign message: crypto/rsa: key size too small " +
		"for PSS signature"

	// Generate new NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Generate new invalid private key and insert into NetworkState
	brokenPrivateKey, err := rsa.GenerateKey(rand.Reader, 128)
	if err != nil {
		t.Fatalf("Failed to generate private key:\n%v", err)
	}
	state.rsaPrivateKey = brokenPrivateKey

	// Update NDF
	err = state.UpdateNdf(testNDF)

	if err == nil || err.Error() != expectedErr {
		t.Errorf("UpdateNdf() did not produce an error when expected."+
			"\n\texpected: %+v\n\treceived: %+v", expectedErr, err)
	}
}

// Tests that GetPrivateKey() returns the correct private key.
func TestNetworkState_GetPrivateKey(t *testing.T) {
	// Generate new private RSA key and NetworkState
	state, expectedPrivateKey, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Get the private
	privateKey := state.GetPrivateKey()

	if !reflect.DeepEqual(privateKey, expectedPrivateKey) {
		t.Errorf("GetPrivateKey() produced an incorrect private key."+
			"\n\texpected: %+v\n\treceived: %+v",
			expectedPrivateKey, privateKey)
	}
}

// Tests that GetRoundMap() returns the correct round StateMap.
func TestNetworkState_GetRoundMap(t *testing.T) {
	// Generate new NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Get the round map
	roundMap := state.GetRoundMap()

	if !reflect.DeepEqual(roundMap, round.NewStateMap()) {
		t.Errorf("GetRoundMap() produced an incorrect round map."+
			"\n\texpected: %+v\n\treceived: %+v",
			round.NewStateMap(), roundMap)
	}
}

// Tests that GetNodeMap() returns the correct node StateMap.
func TestNetworkState_GetNodeMap(t *testing.T) {
	// Generate new NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Get the round map
	nodeMap := state.GetNodeMap()

	if !reflect.DeepEqual(nodeMap, node.NewStateMap()) {
		t.Errorf("GetNodeMap() produced an incorrect node map."+
			"\n\texpected: %+v\n\treceived: %+v",
			node.NewStateMap(), nodeMap)
	}
}

// Tests that NodeUpdateNotification() correctly sends an update to the update
// channel and that GetNodeUpdateChannel() receives and returns it.
func TestNetworkState_NodeUpdateNotification(t *testing.T) {
	// Test values
	testNun := node.UpdateNotification{
		Node:         id.NewIdFromUInt(mrand.Uint64(), id.Node, t),
		FromActivity: current.NOT_STARTED,
		ToActivity:   current.WAITING,
	}

	// Generate new NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	go func() {
		err = state.SendUpdateNotification(testNun)
		if err != nil {
			t.Errorf("NodeUpdateNotification() produced an unexpected error:"+
				"\n%+v", err)
		}
	}()

	nodeUpdateNotifier := state.GetNodeUpdateChannel()

	select {
	case testUpdate := <-nodeUpdateNotifier:
		if !reflect.DeepEqual(testUpdate, testNun) {
			t.Errorf("GetNodeUpdateChannel() received the wrong "+
				"NodeUpdateNotification.\n\texpected: %v\n\t received: %v",
				testNun, testUpdate)
		}
	case <-time.After(50 * time.Millisecond):
		t.Error("Failed to receive node update.")
	}
}

// Tests that NodeUpdateNotification() correctly produces and error when the
// channel buffer is already filled.
func TestNetworkState_NodeUpdateNotification_Error(t *testing.T) {
	// Test values
	testNun := node.UpdateNotification{
		Node:         id.NewIdFromUInt(mrand.Uint64(), id.Node, t),
		FromActivity: current.NOT_STARTED,
		ToActivity:   current.WAITING,
	}
	expectedError := errors.New("Could not send update notification")

	// Generate new NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}

	// Fill buffer
	for i := 0; i < updateBufferLength; i++ {
		state.update <- testNun
	}

	go func() {
		err = state.SendUpdateNotification(testNun)
		if strings.Compare(err.Error(), expectedError.Error()) != 0 {
			t.Errorf("NodeUpdateNotification() did not produce an error "+
				"when the channel buffer is full.\n\texpected: %v\n\treceived: %v",
				expectedError, err)
		}
	}()

	time.Sleep(1 * time.Second)
}

// generateTestNetworkState returns a newly generated NetworkState and private
// key. Errors created by generating the key or NetworkState are returned.
func generateTestNetworkState() (*NetworkState, *rsa.PrivateKey, error) {
	// Generate new private RSA key
	keyPath := testkeys.GetNodeKeyPath()
	keyData := testkeys.LoadFromPath(keyPath)

	privKey, err := rsa.LoadPrivateKeyFromPem(keyData)
	if err != nil {
		return nil, privKey, errors.Errorf("Could not load public key: %v", err)
	}

	// Generate new NetworkState using the private key
	state, err := NewState(privKey, 8, "", region.GetCountryBins())
	if err != nil {
		return state, privKey, fmt.Errorf("NewState() produced an unexpected error:\n+%v", err)
	}

	return state, privKey, nil
}

// Tests that IncrementRoundID() increments the ID correctly.
func TestNetworkState_IncrementRoundID(t *testing.T) {
	testID := uint64(9843)
	var err error
	PermissioningDb, _, err = NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}

	testPath := "testRoundID.txt"
	incrementAmount := uint64(10)
	testState := NetworkState{}
	testState.roundID = id.Round(testID)

	defer func() {
		err := os.RemoveAll(testPath)
		if err != nil {
			t.Fatalf("%+v", err)
		}
	}()

	for i := uint64(0); i < incrementAmount; i++ {
		oldID, err := testState.IncrementRoundID()
		if err != nil {
			t.Errorf("IncrementRoundID() produced an unexpected error on "+
				"index %d: %+v", i, err)
		}

		// Test that the correct old ID was returned
		if oldID != id.Round(testID+i) {
			t.Errorf("IncrementRoundID() did not return the correct old ID."+
				"\n\texpected: %+v\n\treceived: %+v", id.Round(testID+i), oldID)
		}
	}
}

// Tests that GetRoundID() returns the correct value.
func TestNetworkState_GetRoundID(t *testing.T) {
	expectedID := id.Round(9843)

	var err error
	PermissioningDb, _, err = NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf(err.Error())
		t.FailNow()
	}
	err = PermissioningDb.UpsertState(&State{
		Key:   RoundIdKey,
		Value: fmt.Sprintf("%d", expectedID),
	})
	if err != nil {
		t.Errorf(err.Error())
	}

	testState := NetworkState{}

	testID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	if expectedID != testID {
		t.Errorf("GetRoundID() returned an incorrect ID."+
			"\n\texpected: %+v\n\treceived: %+v", expectedID, testID)
	}
}

// Tests that CreateDisabledNodes() correctly generates a disabledNodes and
// saves it to NetworkState.
func TestNetworkState_CreateDisabledNodes(t *testing.T) {
	// Get test data
	testData, stateMap, expectedStateSet := generateIdLists(3, t)
	state := &NetworkState{nodes: stateMap, pruneList: make(map[id.ID]interface{})}
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

	err = state.CreateDisabledNodes(testPath, 33*time.Millisecond)
	if err != nil {
		t.Errorf("CreateDisabledNodes() generated an unexpected error."+
			"\n\texpected: %v\n\treceived: %v", nil, err)
	}

	if !reflect.DeepEqual(state.disabledNodesStates.nodes, expectedStateSet) {
		t.Errorf("CreateDisabledNodes() did not return the correct Set."+
			"\n\texpected: %v\n\treceived: %v",
			expectedStateSet, state.disabledNodesStates.nodes)
	}
}

// Tests that CreateDisabledNodes() gets an error when given an invalid path.
func TestNetworkState_CreateDisabledNodes_FileError(t *testing.T) {
	// Generate new NetworkState
	state, _, err := generateTestNetworkState()
	if err != nil {
		t.Fatalf("%+v", err)
	}
	testPath := "testDisabledNodesList.txt"
	expectedErr := "Skipping polling of disabled node ID list file; error " +
		"while accessing file: open " + testPath + ": The system cannot find " +
		"the file specified."

	err = state.CreateDisabledNodes(testPath, 33*time.Millisecond)
	if err == nil {
		t.Errorf("CreateDisabledNodes() did not error on invalid path."+
			"\n\texpected: %v\n\treceived: %v", expectedErr, err)
	}

	if state.disabledNodesStates != nil {
		t.Errorf("CreateDisabledNodes() did not set a nil object on error."+
			"\n\texpected: %v\n\treceived: %v", nil, state.disabledNodesStates)
	}
}

// Tests that StartPollDisabledNodes() correctly stops looping when triggering
// the quit channel.
func TestNetworkState_StartPollDisabledNodes(t *testing.T) {
	// Get test data
	state := &NetworkState{disabledNodesStates: &disabledNodes{
		nodes:    nil,
		path:     "testDisabledNodesList.txt",
		interval: 33 * time.Millisecond,
	}}

	result := make(chan bool)
	quit := make(chan struct{})

	go func() {
		state.StartPollDisabledNodes(quit)
		result <- true
	}()

	quit <- struct{}{}

	select {
	case <-result:
		return
	case <-time.After(500 * time.Millisecond):
		t.Errorf("StartPollDisabledNodes() did not correctly stop when kill command sent.")
	}
}
