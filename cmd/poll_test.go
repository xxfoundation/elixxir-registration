////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"bytes"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/testkeys"
	"sync/atomic"
	"testing"
	"time"
)

// TestFunc: Gets permissioning server test key
func getTestKey() *rsa.PrivateKey {
	permKeyBytes, _ := utils.ReadFile(testkeys.GetCAKeyPath())

	testPermissioningKey, _ := rsa.LoadPrivateKeyFromPem(permKeyBytes)
	return testPermissioningKey
}

// Happy path
func TestRegistrationImpl_Poll(t *testing.T) {
	testID := id.NewNodeFromUInt(0, t)
	testString := "test"
	// Start registration server
	testParams.KeyPath = testkeys.GetCAKeyPath()
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf("Unable to start registration: %+v", err)
	}
	atomic.CompareAndSwapUint32(impl.NdfReady, 0, 1)

	err = impl.State.UpdateNdf(&ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:        "420",
			TlsCertificate: "",
		},
	})

	// Make a simple auth object that will pass the checks
	testHost, _ := connect.NewHost(testID.String(), testString,
		make([]byte, 0), false, true)
	testAuth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}
	testMsg := &pb.PermissioningPoll{
		Full: &pb.NDFHash{
			Hash: []byte(testString)},
		Partial: &pb.NDFHash{
			Hash: []byte(testString),
		},
		LastUpdate: 0,
		Activity:   uint32(current.WAITING),
		Error:      nil,
	}

	err = impl.State.AddRoundUpdate(
		&pb.RoundInfo{
			ID:    1,
			State: uint32(states.PRECOMPUTING),
		})

	if err != nil {
		t.Errorf("Could not add round update: %s", err)
	}

	err = impl.State.GetNodeMap().AddNode(testID, "")

	if err != nil {
		t.Errorf("Could nto add node: %s", err)
	}

	response, err := impl.Poll(testMsg, testAuth)
	if err != nil {
		t.Errorf("Unexpected error polling: %+v", err)
	}
	if len(response.GetUpdates()) != 1 {
		t.Errorf("Expected round updates to return!")
	}
	if response.GetUpdates()[0].State != uint32(states.PRECOMPUTING) {
		t.Errorf("Expected round to update to PRECOMP! Got %+v", response.GetUpdates())
	}

	// Shutdown registration
	impl.Comms.Shutdown()
}

// Error path: Ndf not ready
func TestRegistrationImpl_PollNoNdf(t *testing.T) {

	// Read in private key
	key, err := utils.ReadFile(testkeys.GetCAKeyPath())
	if err != nil {
		t.Errorf("failed to read key at %+v: %+v",
			testkeys.GetCAKeyPath(), err)
	}

	pk, err := rsa.LoadPrivateKeyFromPem(key)
	if err != nil {
		t.Errorf("Failed to parse permissioning server key: %+v. "+
			"PermissioningKey is %+v", err, pk)
	}
	// Start registration server
	ndfReady := uint32(0)
	state, err := storage.NewState(pk)
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	impl := &RegistrationImpl{
		State:    state,
		NdfReady: &ndfReady,
	}

	_, err = impl.Poll(nil, nil)
	if err == nil || err.Error() != ndf.NO_NDF {
		t.Errorf("Unexpected error polling: %+v", err)
	}
}

// Error path: Failed auth
func TestRegistrationImpl_PollFailAuth(t *testing.T) {
	testString := "test"

	// Start registration server
	ndfReady := uint32(1)
	state, err := storage.NewState(getTestKey())
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	impl := RegistrationImpl{
		State:    state,
		NdfReady: &ndfReady,
	}

	err = impl.State.UpdateNdf(&ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:        "420",
			TlsCertificate: "",
		},
	})

	// Make a simple auth object that will fail the checks
	testHost, _ := connect.NewHost(testString, testString,
		make([]byte, 0), false, true)
	testAuth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          testHost,
	}

	_, err = impl.Poll(nil, testAuth)
	if err == nil || err.Error() != connect.AuthError(testAuth.Sender.GetId()).Error() {
		t.Errorf("Unexpected error polling: %+v", err)
	}
}

//Happy path
func TestRegistrationImpl_PollNdf(t *testing.T) {

	//Create database
	storage.PermissioningDb = storage.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")

	//Create reg codes and populate the database
	infos := make([]node.Info, 0)
	infos = append(infos, node.Info{RegCode: "BBBB"},
		node.Info{RegCode: "CCCC"},
		node.Info{RegCode: "DDDD"})
	storage.PopulateNodeRegistrationCodes(infos)

	RegParams = testParams
	udbId := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	RegParams.udbId = udbId
	RegParams.minimumNodes = 3

	// Start registration server
	impl, err := StartRegistration(RegParams)
	if err != nil {
		t.Errorf(err.Error())
	}

	beginScheduling := make(chan struct{}, 1)
	go func() {
		err = impl.nodeRegistrationCompleter(beginScheduling)
		if err != nil {
			t.Errorf(err.Error())
		}
	}()

	//Register 1st node
	err = impl.RegisterNode([]byte("B"), nodeAddr, string(nodeCert),
		"0.0.0.0:7900", string(gatewayCert), "BBBB")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	//Register 2nd node
	err = impl.RegisterNode([]byte("C"), "0.0.0.0:6901", string(nodeCert),
		"0.0.0.0:7901", string(gatewayCert), "CCCC")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	//Register 3rd node
	err = impl.RegisterNode([]byte("D"), "0.0.0.0:6902", string(nodeCert),
		"0.0.0.0:7902", string(gatewayCert), "DDDD")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	//wait for registration to complete
	select {
	case <-time.NewTimer(100 * time.Millisecond).C:
		t.Errorf("Node registration never completed")
		t.FailNow()
	case <-beginScheduling:
	}

	observedNDFBytes, err := impl.PollNdf(nil, &connect.Auth{})
	if err != nil {
		t.Errorf("failed to update ndf: %v", err)
	}

	observedNDF, _, err := ndf.DecodeNDF(string(observedNDFBytes))
	if err != nil {
		t.Errorf("Could not decode ndf: %v\nNdf output: %s", err,
			string(observedNDFBytes))
	}
	if bytes.Compare(observedNDF.UDB.ID, udbId) != 0 {
		t.Errorf("Failed to set udbID. Expected: %v, \nRecieved: %v, \nNdf: %+v",
			udbId, observedNDF.UDB.ID, observedNDF)
	}

	if observedNDF.Registration.Address != permAddr {
		t.Errorf("Failed to set registration address. Expected: %v \n Recieved: %v",
			permAddr, observedNDF.Registration.Address)
	}
	expectedNodeIDs := make([][]byte, 0)
	expectedNodeIDs = append(expectedNodeIDs, []byte("B"), []byte("C"), []byte("D"))
	for i := range observedNDF.Nodes {
		if bytes.Compare(expectedNodeIDs[i], observedNDF.Nodes[i].ID) != 0 {
			t.Errorf("Could not build node %d's, id: Expected: %v \n Recieved: %v", i,
				expectedNodeIDs, observedNDF.Nodes[i].ID)
		}
	}

	//Shutdown node comms
	impl.Comms.Shutdown()
}

//Error  path
func TestRegistrationImpl_PollNdf_NoNDF(t *testing.T) {
	//Create database
	storage.PermissioningDb = storage.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")

	//Create reg codes and populate the database
	infos := make([]node.Info, 0)
	infos = append(infos, node.Info{RegCode: "BBBB"},
		node.Info{RegCode: "CCCC"},
		node.Info{RegCode: "DDDD"})
	storage.PopulateNodeRegistrationCodes(infos)
	RegParams = testParams
	//Setup udb configurations
	udbId := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	RegParams.udbId = udbId
	RegParams.minimumNodes = 3

	// Start registration server
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf(err.Error())
	}

	beginScheduling := make(chan struct{}, 1)

	go impl.nodeRegistrationCompleter(beginScheduling)

	//Register 1st node
	err = impl.RegisterNode([]byte("B"), nodeAddr, string(nodeCert),
		"0.0.0.0:7900", string(gatewayCert), "BBBB")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	//Make a client ndf hash that is not up to date
	clientNdfHash := []byte("test")

	_, err = impl.PollNdf(clientNdfHash, &connect.Auth{})
	if err == nil {
		t.Error("Expected error path, should not have an ndf ready")
	}
	if err != nil && err.Error() != ndf.NO_NDF {
		t.Errorf("Expected correct error message: %+v", err)
	}

	//Shutdown registration
	impl.Comms.Shutdown()
}
