////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"bytes"
	"fmt"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/comms/testutils"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
	"gitlab.com/elixxir/registration/testkeys"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/region"
	"gitlab.com/xx_network/primitives/utils"
	"sync"
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
func TestRegistrationImpl_Poll_NDF(t *testing.T) {

	// Create a database
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Fatalf("Failed to create new database: %+v", err)
	}

	testID := id.NewIdFromUInt(0, id.Node, t)
	testString := "test"
	// Start registration server
	testParams.KeyPath = testkeys.GetCAKeyPath()
	testParams.WhitelistedIdsPath = testkeys.GetPreApprovedPath()
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf("Unable to start registration: %+v", err)
	}
	atomic.CompareAndSwapUint32(impl.NdfReady, 0, 1)

	impl.State.UpdateInternalNdf(&ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:        "420",
			TlsCertificate: "",
		},
		Gateways: []ndf.Gateway{
			{ID: id.NewIdFromUInt(0, id.Gateway, t).Bytes()},
		},
		Nodes: []ndf.Node{
			{ID: id.NewIdFromUInt(0, id.Node, t).Bytes()},
		},
	})
	err = impl.State.UpdateOutputNdf()
	if err != nil {
		t.Fatalf("Failed to update ndf: %+v", err)
	}

	// Make a simple auth object that will pass the checks
	testHost, _ := impl.Comms.AddHost(testID, testString,
		make([]byte, 0), connect.GetDefaultHostParams())

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
		LastUpdate:     0,
		Activity:       uint32(current.WAITING),
		Error:          nil,
		GatewayVersion: "1.1.0",
		ServerVersion:  "1.1.0",
		GatewayAddress: "",
	}
	err = impl.State.AddRoundUpdate(
		&pb.RoundInfo{
			ID:         1,
			State:      uint32(states.PRECOMPUTING),
			Timestamps: make([]uint64, states.FAILED),
		})

	if err != nil {
		t.Errorf("Could not add round update: %s", err)
	}

	err = impl.State.GetNodeMap().AddNode(testID, "", "", "", 0)

	if err != nil {
		t.Errorf("Could nto add node: %s", err)
	}

	n := impl.State.GetNodeMap().GetNode(testID)
	n.SetConnectivity(node.PortSuccessful)

	impl.params.disablePing = true

	response, err := impl.Poll(testMsg, testAuth)
	if err != nil {
		t.Errorf("Unexpected error polling: %+v", err)
	}
	time.Sleep(100 * time.Millisecond)

	if response.FullNDF == nil {
		t.Errorf("No NDF provided")
	}

	// Shutdown registration
	impl.Comms.Shutdown()
}

func TestRegistrationImpl_Poll_Round(t *testing.T) {
	testID := id.NewIdFromUInt(0, id.Node, t)
	testString := "test"
	// Start registration server
	testParams.KeyPath = testkeys.GetCAKeyPath()
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf("Unable to start registration: %+v", err)
	}
	atomic.CompareAndSwapUint32(impl.NdfReady, 0, 1)

	impl.State.UpdateInternalNdf(&ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:        "420",
			TlsCertificate: "",
		},
		Gateways: []ndf.Gateway{
			{ID: id.NewIdFromUInt(0, id.Gateway, t).Bytes()},
		},
		Nodes: []ndf.Node{
			{ID: id.NewIdFromUInt(0, id.Node, t).Bytes()},
		},
	})
	err = impl.State.UpdateOutputNdf()
	if err != nil {
		t.Fatalf("Failed to update output ndf: %+v", err)
	}

	// Make a simple auth object that will pass the checks
	testHost, _ := impl.Comms.AddHost(testID, testString,
		make([]byte, 0), connect.GetDefaultHostParams())

	testAuth := &connect.Auth{
		IsAuthenticated: true,
		Sender:          testHost,
	}
	testMsg := &pb.PermissioningPoll{
		Full: &pb.NDFHash{
			Hash: impl.State.GetFullNdf().GetHash(),
		},
		Partial: &pb.NDFHash{
			Hash: []byte(testString),
		},
		LastUpdate:     0,
		Activity:       uint32(current.WAITING),
		Error:          nil,
		GatewayVersion: "1.1.0",
		ServerVersion:  "1.1.0",
		GatewayAddress: "",
	}
	err = impl.State.AddRoundUpdate(
		&pb.RoundInfo{
			ID:         1,
			State:      uint32(states.PRECOMPUTING),
			Timestamps: make([]uint64, states.FAILED),
		})
	time.Sleep(100 * time.Millisecond)

	if err != nil {
		t.Errorf("Could not add round update: %s", err)
	}

	err = impl.State.GetNodeMap().AddNode(testID, "", "", "", 0)

	if err != nil {
		t.Errorf("Could nto add node: %s", err)
	}

	n := impl.State.GetNodeMap().GetNode(testID)
	n.SetConnectivity(node.PortSuccessful)

	impl.params.disablePing = true

	response, err := impl.Poll(testMsg, testAuth)
	if err != nil {
		t.Errorf("Unexpected error polling: %+v", err)
	}
	time.Sleep(100 * time.Millisecond)

	if len(response.GetUpdates()) != 1 {
		t.Errorf("Expected round updates to return!")
	}
	fmt.Println(response.GetUpdates()[0])
	if response.GetUpdates()[0].State != uint32(states.PRECOMPUTING) {
		t.Errorf("Expected round to update to PRECOMP! Got %+v", response.GetUpdates())
	}

	// Shutdown registration
	impl.Comms.Shutdown()
}

/*// Error path: Ndf not ready
func TestRegistrationImpl_PollNoNdf(t *testing.T) {

	pk, err := testutils.LoadPrivateKeyTesting(t)
	if err != nil {
		t.Errorf("Failed to parse permissioning server key: %+v. "+
			"PermissioningKey is %+v", err, pk)
	}
	// Start registration server
	ndfReady := uint32(0)
	state, err := storage.NewState(pk, 8, "")
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}
	testVersion, _ := version.ParseVersion("0.0.0")
	impl := &RegistrationImpl{
		State:    state,
		NdfReady: &ndfReady,
		params: &Params{
			minGatewayVersion: testVersion,
			minServerVersion:  testVersion,
		},
	}

	dummyMessage := &pb.PermissioningPoll{}

	_, err = impl.Poll(dummyMessage, nil)
	if err == nil || err.Error() != ndf.NO_NDF {
		t.Errorf("Unexpected error polling: %+v", err)
	}
}*/

// Happy path
func TestRegistrationImpl_PollNdf(t *testing.T) {
	//Create database
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("",
		"", "", "", "")
	if err != nil {
		t.Errorf("%+v", err)
	}

	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	//Create reg codes and populate the database
	infos := []node.Info{
		{RegCode: "AAAA", Order: "CR"},
		{RegCode: "BBBB", Order: "GB"},
		{RegCode: "CCCC", Order: "BF"},
		{RegCode: "DDDD", Order: "BF"},
	}
	storage.PopulateNodeRegistrationCodes(infos)

	RegParams = testParams
	udbId := id.NewIdFromUInt(5, id.User, t)
	RegParams.udbId = udbId.Marshal()
	RegParams.minimumNodes = 3
	RegParams.disableNDFPruning = true
	// Start registration server
	impl, err := StartRegistration(RegParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	defer impl.Comms.Shutdown()

	var l sync.Mutex
	go func() {
		l.Lock()
		defer l.Unlock()
		//Register 1st node
		testSalt := []byte("testtesttesttesttesttesttesttest")
		testSalt2 := []byte("testtesttesttesttesttesttesttesc")
		testSalt3 := []byte("testtesttesttesttesttesttesttesd")
		err = impl.RegisterNode(testSalt,
			nodeAddr, string(nodeCert),
			"0.0.0.0:7900", string(gatewayCert), "BBBB")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
		}
		//Register 2nd node
		err = impl.RegisterNode(testSalt2,
			"0.0.0.0:6901", string(nodeCert),
			"0.0.0.0:7901", string(gatewayCert), "CCCC")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
		}
		//Register 3rd node
		err = impl.RegisterNode(testSalt3,
			"0.0.0.0:6902", string(nodeCert),
			"0.0.0.0:7902", string(gatewayCert), "DDDD")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
		}
	}()

	//wait for registration to complete
	select {
	case <-time.NewTimer(1000 * time.Millisecond).C:
		t.Errorf("Node registration never completed")
		t.FailNow()
	case <-impl.beginScheduling:
	}

	err = impl.State.UpdateOutputNdf()
	if err != nil {
		t.Fatalf("Failed to update output ndf: %+v", err)
	}

	l.Lock()
	observedNDFBytes, err := impl.PollNdf(nil)
	l.Unlock()
	if err != nil {
		t.Errorf("failed to update ndf: %v", err)
	}

	observedNDF, err := ndf.Unmarshal(observedNDFBytes.Ndf)
	if err != nil {
		t.Errorf("Could not decode ndf: %v\nNdf output: %s", err,
			string(observedNDFBytes.Ndf))
	}

	if bytes.Compare(observedNDF.UDB.ID, udbId.Marshal()) != 0 {
		t.Errorf("Failed to set udbID. Expected: %v, \nRecieved: %v, \nNdf: %+v",
			udbId, observedNDF.UDB.ID, observedNDF)
	}

	if observedNDF.Registration.Address != permAddr {
		t.Errorf("Failed to set registration address. Expected: %v \n Recieved: %v",
			permAddr, observedNDF.Registration.Address)
	}

	if len(observedNDF.Nodes) != 3 {
		t.Errorf("Did not receive expected node count.\n\tExpected: %d\n\tReceived: %d\n", 3, len(observedNDF.Nodes))
	}
}

// Error  path
func TestRegistrationImpl_PollNdf_NoNDF(t *testing.T) {
	//Create database
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf("%+v", err)
	}

	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	//Create reg codes and populate the database
	infos := []node.Info{
		{RegCode: "AAAA", Order: "CR"},
		{RegCode: "BBBB", Order: "GB"},
		{RegCode: "CCCC", Order: "BF"},
	}
	storage.PopulateNodeRegistrationCodes(infos)
	RegParams = testParams
	//Setup udb configurations
	udbId := id.NewIdFromUInt(5, id.User, t)
	RegParams.udbId = udbId.Marshal()
	RegParams.minimumNodes = 3
	RegParams.disableNDFPruning = true
	// Start registration server
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	//Shutdown registration
	defer impl.Comms.Shutdown()

	//Register 1st node
	testSalt := []byte("testtesttesttesttesttesttesttest")
	err = impl.RegisterNode(testSalt, nodeAddr, string(nodeCert),
		"0.0.0.0:7900", string(gatewayCert), "BBBB")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	//Make a client ndf hash that is not up to date
	clientNdfHash := []byte("test")

	_, err = impl.PollNdf(clientNdfHash)
	if err == nil {
		t.Error("Expected error path, should not have an ndf ready")
	}
	if err != nil && err.Error() != ndf.NO_NDF {
		t.Errorf("Expected correct error message: %+v", err)
	}
}

func TestPoll_BannedNode(t *testing.T) {
	//Create database
	var err error

	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf("%+v", err)
	}

	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	testID := id.NewIdFromUInt(0, id.Node, t)
	testString := "test"
	// Start registration server
	testParams.KeyPath = testkeys.GetCAKeyPath()
	testParams.disableNDFPruning = true
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf("Unable to start registration: %+v", err)
	}
	defer impl.Comms.Shutdown()
	atomic.CompareAndSwapUint32(impl.NdfReady, 0, 1)

	impl.State.UpdateInternalNdf(&ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:        "420",
			TlsCertificate: "",
		},
	})
	err = impl.State.UpdateOutputNdf()
	if err != nil {
		t.Fatalf("Failed to update output ndf: %+v", err)
	}

	// Make a simple auth object that will pass the checks
	testHost, _ := connect.NewHost(testID, testString,
		make([]byte, 0), connect.GetDefaultHostParams())
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
			ID:         1,
			State:      uint32(states.PRECOMPUTING),
			Timestamps: make([]uint64, states.FAILED),
		})

	if err != nil {
		t.Errorf("Could not add round update: %s", err)
	}

	err = impl.State.GetNodeMap().AddNode(testID, "", "", "", 0)
	if err != nil {
		t.Errorf("Could nto add node: %s", err)
	}

	_, err = impl.State.GetNodeMap().GetNode(testID).Ban()
	if err != nil {
		t.Errorf(err.Error())
	}

	_, err = impl.Poll(testMsg, testAuth)
	if err != nil {
		return
	}

	t.Errorf("Expected error state: Node with out of network status should return an error")
}

// TODO: more work needs to be done to get this testable (making timeout a config option, etc)
//func TestPoll_CheckPortForwarding(t *testing.T) {
//	testID := id.NewIdFromUInt(0, id.Node, t)
//	testString := "test"
//	// Start registration server
//	testParams.KeyPath = testkeys.GetCAKeyPath()
//	impl, err := StartRegistration(testParams, nil)
//	if err != nil {
//		t.Errorf("Unable to start registration: %+v", err)
//	}
//	atomic.CompareAndSwapUint32(impl.NdfReady, 0, 1)
//
//	err = impl.State.UpdateInternalNdf(&ndf.NetworkDefinition{
//		Registration: ndf.Registration{
//			Address:        "420",
//			TlsCertificate: "",
//		},
//		Gateways: []ndf.Gateway{
//			{ID: id.NewIdFromUInt(0, id.Gateway, t).Bytes()},
//		},
//		Nodes: []ndf.Node{
//			{ID: id.NewIdFromUInt(0, id.Node, t).Bytes()},
//		},
//	})
//
//	// Make a simple auth object that will pass the checks
//	testHost, _ := connect.NewHost(testID, testString,
//		make([]byte, 0), false, true)
//	testAuth := &connect.Auth{
//		IsAuthenticated: true,
//		Sender:          testHost,
//	}
//	testMsg := &pb.PermissioningPoll{
//		Full: &pb.NDFHash{
//			Hash: []byte(testString)},
//		Partial: &pb.NDFHash{
//			Hash: []byte(testString),
//		},
//		LastUpdate:     0,
//		Activity:       uint32(current.WAITING),
//		Error:          nil,
//		GatewayVersion: "1.1.0",
//		ServerVersion:  "1.1.0",
//	}
//
//	err = impl.State.AddRoundUpdate(
//		&pb.RoundInfo{
//			ID:    1,
//			State: uint32(states.PRECOMPUTING),
//		})
//
//	if err != nil {
//		t.Errorf("Could not add round update: %s", err)
//	}
//
//	err = impl.State.GetNodeMap().AddNode(testID, "", "", "")
//
//	if err != nil {
//		t.Errorf("Could nto add node: %s", err)
//	}
//
//	_, err = impl.Poll(testMsg, testAuth, "0.0.0.0")
//	if err != nil {
//		t.Errorf("Unexpected error polling: %+v", err)
//	}
//
//
//	n := impl.State.GetNodeMap().GetNode(testID)
//	if n.GetConnectivity() != node.PortFailed {
//		t.Errorf("Failed to set node connectivity as expected!" +
//			"\n\tExpected: %v" +
//			"\n\tReceived: %v", node.PortFailed, n.GetConnectivity())
//	}
//
//
//	// Shutdown registration
//	impl.Comms.Shutdown()
//}

// Check that checkVersion() correctly determines the message versions to be
// compatible with the required versions.
func TestCheckVersion(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "1.5.6",
		GatewayVersion: "1.4.5",
	}

	requiredServer, _ := version.ParseVersion("1.3.2")
	requiredGateway, _ := version.ParseVersion("1.3.2")
	p := &Params{
		minGatewayVersion: requiredGateway,
		minServerVersion:  requiredServer,
	}

	err := checkVersion(p, testMsg)
	if err != nil {
		t.Errorf("checkVersion() unexpectedly errored: %+v", err)
	}
}

// Check that checkVersion() skips checking versions when the gateway version is
// blank.
func TestCheckVersion_EmptyVersions(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "1.5.6",
		GatewayVersion: "",
	}

	requiredServer, _ := version.ParseVersion("1.3.2")
	requiredGateway, _ := version.ParseVersion("1.3.2")

	p := &Params{
		minGatewayVersion: requiredGateway,
		minServerVersion:  requiredServer,
	}

	err := checkVersion(p, testMsg)
	if err != nil {
		t.Errorf("checkVersion() unexpectedly errored on empty version "+
			"strings: %+v", err)
	}
}

// Check that checkVersion() correctly determines the message versions to be
// compatible with the required version when they are equal.
func TestCheckVersion_Edge(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "1.3.2b",
		GatewayVersion: "1.3.2c",
	}

	requiredServer, _ := version.ParseVersion("1.3.2")
	requiredGateway, _ := version.ParseVersion("1.3.2")
	p := &Params{
		minGatewayVersion: requiredGateway,
		minServerVersion:  requiredServer,
	}

	err := checkVersion(p, testMsg)
	if err != nil {
		t.Errorf("checkVersion() unexpectedly errored: %+v", err)
	}
}

// Check that checkVersion() returns an error if the gateway version cannot be
// parsed.
func TestCheckVersion_ParseErrorGateway(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "1.5.6",
		GatewayVersion: "1.a.5",
	}

	requiredServer, _ := version.ParseVersion("1.3.2")
	requiredGateway, _ := version.ParseVersion("1.3.2")

	p := &Params{
		minGatewayVersion: requiredGateway,
		minServerVersion:  requiredServer,
	}

	err := checkVersion(p, testMsg)
	if err == nil {
		t.Errorf("checkVersion() did not error on invalid gateway version.")
	}
}

// Check that checkVersion() returns an error if the server version cannot be
// parsed.
func TestCheckVersion_ParseErrorServer(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "afw.5.6",
		GatewayVersion: "1.4.5",
	}

	requiredServer, _ := version.ParseVersion("1.3.2")
	requiredGateway, _ := version.ParseVersion("1.3.2")

	p := &Params{
		minGatewayVersion: requiredGateway,
		minServerVersion:  requiredServer,
	}

	err := checkVersion(p, testMsg)
	if err == nil {
		t.Errorf("checkVersion() did not error on invalid server version.")
	}
}

// Check that checkVersion() returns an error for an incompatible gateway
// version.
func TestCheckVersion_InvalidVersionGateway(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "1.5.6",
		GatewayVersion: "1.4.5",
	}

	requiredServer, _ := version.ParseVersion("1.3.2")
	requiredGateway, _ := version.ParseVersion("4.3.2")

	expectedError := "The gateway version \"" + testMsg.GatewayVersion +
		"\" is incompatible with the required version \"" +
		requiredGateway.String() + "\"."

	p := &Params{
		minGatewayVersion: requiredGateway,
		minServerVersion:  requiredServer,
	}

	err := checkVersion(p, testMsg)
	if err != nil && err.Error() != expectedError {
		t.Errorf("checkVersion() did not produce the correct error on "+
			"incompatible gateway version.\n\texpected: %+v\n\treceived: %+v",
			expectedError, err)
	} else if err == nil {
		t.Errorf("checkVersion() did not error on incompatible gateway " +
			"version.")
	}
}

// Check that checkVersion() returns an error for an incompatible server
// version.
func TestCheckVersion_InvalidVersionServer(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "1.5.6",
		GatewayVersion: "1.4.5",
	}

	requiredServer, _ := version.ParseVersion("1.15.2")
	requiredGateway, _ := version.ParseVersion("1.3.2")

	expectedError := "The server version \"" + testMsg.ServerVersion +
		"\" is incompatible with the required version \"" +
		requiredServer.String() + "\"."

	p := &Params{
		minGatewayVersion: requiredGateway,
		minServerVersion:  requiredServer,
	}

	err := checkVersion(p, testMsg)
	if err != nil && err.Error() != expectedError {
		t.Errorf("checkVersion() did not produce the correct error on "+
			"incompatible server version.\n\texpected: %+v\n\treceived: %+v",
			expectedError, err)
	} else if err == nil {
		t.Errorf("checkVersion() did not error on incompatible server " +
			"version.")
	}
}

// Check that checkVersion() returns an error for an incompatible gateway
// version when both gateway and server are of incompatible versions.
func TestCheckVersion_InvalidVersionGatewayAndServer(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "0.6.7b",
		GatewayVersion: "1.0.a",
	}

	requiredServer, _ := version.ParseVersion("1.1.0")
	requiredGateway, _ := version.ParseVersion("1.1.0")

	expectedError := "The gateway version \"" + testMsg.GatewayVersion +
		"\" is incompatible with the required version \"" +
		requiredGateway.String() + "\"."

	p := &Params{
		minGatewayVersion: requiredGateway,
		minServerVersion:  requiredServer,
	}

	err := checkVersion(p, testMsg)
	if err != nil && err.Error() != expectedError {
		t.Errorf("checkVersion() did not produce the correct error on "+
			"incompatible gateway version.\n\texpected: %+v\n\treceived: %+v",
			expectedError, err)
	} else if err == nil {
		t.Errorf("checkVersion() did not error on incompatible gateway " +
			"version.")
	}
}

/*func TestUpdateNDF(t *testing.T) {
	testID := id.NewIdFromUInt(0, id.Node, t)
	testString := "test"
	// Start registration server
	testParams.KeyPath = testkeys.GetCAKeyPath()
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf("Unable to start registration: %+v", err)
	}
	atomic.CompareAndSwapUint32(impl.NdfReady, 0, 1)

	err = impl.State.UpdateInternalNdf(&ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:        "420",
			TlsCertificate: "",
		},
		Gateways: []ndf.Gateway{
			{ID: id.NewIdFromUInt(0, id.Gateway, t).Bytes()},
		},
		Nodes: []ndf.Node{
			{ID: id.NewIdFromUInt(0, id.Node, t).Bytes()},
		},
	})

	// Make a simple auth object that will pass the checks
	testHost, _ := connect.NewHost(testID, testString, make([]byte, 0), false, true)
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
		LastUpdate:     0,
		Activity:       uint32(current.WAITING),
		Error:          nil,
		GatewayVersion: "0.0.0",
		GatewayAddress: "0.0.0.0",
		ServerVersion:  "0.0.0",
		ServerPort:     45622,
	}

	err = impl.State.AddRoundUpdate(
		&pb.RoundInfo{
			ID:    1,
			State: uint32(states.PRECOMPUTING),
		})

	if err != nil {
		t.Errorf("Could not add round update: %s", err)
	}

	err = impl.State.GetNodeMap().AddNode(testID, "", "", "")

	if err != nil {
		t.Errorf("Could nto add node: %s", err)
	}

	nid := testAuth.Sender.GetId()

	err = impl.updateNDF(*nid, impl.State.GetNodeMap().GetNode(nid), testMsg, "0.0.0.0")

	if err != nil {
		t.Errorf("updateNDF() unexpectedly errored: %+v", err)
	}

	expectedNodeAddress := strings.Join([]string{"0.0.0.0", strconv.Itoa(int(testMsg.ServerPort))}, ":")

	newNodeAddress := impl.State.GetFullNdf().Get().Nodes[0].Address
	if newNodeAddress != expectedNodeAddress {
		t.Errorf("updateNDF() did not update node address correctly."+
			"\n\texpected: %#v\n\treceived: %#v", expectedNodeAddress, newNodeAddress)
	}

	expectedGatewayAddress := testMsg.GatewayAddress

	newGatewayAddress := impl.State.GetFullNdf().Get().Gateways[0].Address
	if newGatewayAddress != expectedGatewayAddress {
		t.Errorf("updateNDF() did not update gateway address correctly."+
			"\n\texpected: %#v\n\treceived: %#v", expectedGatewayAddress, newGatewayAddress)
	}

	//Kill the connections for the next test
	impl.Comms.Shutdown()

}*/

// Tests that updateNdfNodeAddr() correctly updates the correct node address.
func TestUpdateNdfNodeAddr(t *testing.T) {
	nID := id.NewIdFromUInt(225, id.Node, t)
	requiredAddr := "1.1.1.1:1234"
	testNDF := &ndf.NetworkDefinition{
		Nodes: []ndf.Node{{
			ID:      id.NewIdFromUInt(0, id.Node, t)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(1, id.Node, t)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(3, id.Node, t)[:],
			Address: "0.0.0.3",
		}},
	}

	testNDF.Nodes[2].ID = nID[:]

	err := updateNdfNodeAddr(nID, requiredAddr, testNDF)

	if err != nil {
		t.Errorf("updateNdfNodeAddr() unexpectedly produced an error: %+v", err)
	}

	if testNDF.Nodes[2].Address != requiredAddr {
		t.Errorf("updateNdfNodeAddr() did not update the node address "+
			"correectly\n\texpected: %+v\n\treceived: %+v",
			requiredAddr, testNDF.Nodes[2].Address)
	}
}

// Tests that updateNdfGatewayAddr() correctly updates the correct gateway
// address.
func TestUpdateNdfGatewayAddr(t *testing.T) {
	gwID := id.NewIdFromUInt(742, id.Gateway, t)
	requiredAddr := "1.1.1.1:1234"
	testNDF := &ndf.NetworkDefinition{
		Gateways: []ndf.Gateway{{
			ID:      id.NewIdFromUInt(0, id.Gateway, t)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(1, id.Gateway, t)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(2, id.Gateway, t)[:],
			Address: "0.0.0.3",
		}},
	}

	testNDF.Gateways[2].ID = gwID[:]

	err := updateNdfGatewayAddr(gwID, requiredAddr, testNDF)

	if err != nil {
		t.Errorf("updateNdfGatewayAddr() unexpectedly produced an error: %+v",
			err)
	}

	if testNDF.Gateways[2].Address != requiredAddr {
		t.Errorf("updateNdfGatewayAddr() did not update the gateway address "+
			"correectly\n\texpected: %+v\n\treceived: %+v",
			requiredAddr, testNDF.Gateways[2].Address)
	}
}

// Tests that updateNdfNodeAddr() correctly updates the correct node address.
func TestUpdateNdfNodeAddr_Error(t *testing.T) {
	nID := id.NewIdFromUInt(225, id.Node, t)
	requiredAddr := "1.1.1.1:1234"
	testNDF := &ndf.NetworkDefinition{
		Nodes: []ndf.Node{{
			ID:      id.NewIdFromUInt(0, id.Node, t)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(1, id.Node, t)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(3, id.Node, t)[:],
			Address: "0.0.0.3",
		}},
	}

	err := updateNdfNodeAddr(nID, requiredAddr, testNDF)

	if err == nil {
		t.Errorf("updateNdfNodeAddr() did not produce an error when the node " +
			"ID doesn't exist.")
	}
}

// Tests that updateNdfGatewayAddr() correctly updates the correct gateway
// address.
func TestUpdateNdfGatewayAddr_Error(t *testing.T) {
	gwID := id.NewIdFromUInt(742, id.Gateway, t)
	requiredAddr := "1.1.1.1:1234"
	testNDF := &ndf.NetworkDefinition{
		Gateways: []ndf.Gateway{{
			ID:      id.NewIdFromUInt(0, id.Gateway, t)[:],
			Address: "0.0.0.1",
		}, {
			ID:      id.NewIdFromUInt(1, id.Gateway, t)[:],
			Address: "0.0.0.2",
		}, {
			ID:      id.NewIdFromUInt(2, id.Gateway, t)[:],
			Address: "0.0.0.3",
		}},
	}

	err := updateNdfGatewayAddr(gwID, requiredAddr, testNDF)

	if err == nil {
		t.Errorf("updateNdfGatewayAddr() did not produce an error when the " +
			"gateway ID doesn't exist.")
	}
}

func TestVerifyError(t *testing.T) {
	nodeCert, err := utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		t.Errorf("Could not get node cert: %+v\n", err)
	}

	nodeKey, err = utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Errorf("Could not get node key: %+v\n", err)
	}

	// Read in private key
	pk, err := testutils.LoadPrivateKeyTesting(t)
	if err != nil {
		t.Errorf("Failed to parse permissioning server key: %+v. "+
			"PermissioningKey is %+v", err, pk)
	}
	// Start registration server
	ndfReady := uint32(0)

	state, err := storage.NewState(pk, 8, "", "", region.GetCountryBins())
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}

	testVersion, _ := version.ParseVersion("0.0.0")
	testManager := connect.NewManagerTesting(t)
	impl := &RegistrationImpl{
		State:    state,
		NdfReady: &ndfReady,
		params: &Params{
			minGatewayVersion: testVersion,
			minServerVersion:  testVersion,
			disableNDFPruning: true,
		},
		Comms: &registration.Comms{
			ProtoComms: &connect.ProtoComms{
				Manager: testManager,
			},
		},
	}

	errNodeId := id.NewIdFromString("node", id.Node, t)
	params := connect.GetDefaultHostParams()
	params.AuthEnabled = false
	_, err = impl.Comms.AddHost(errNodeId, "0.0.0.0:8000", nodeCert, params)
	if err != nil {
		t.Error("Failed to add host")
	}

	errMsg := &pb.RoundError{
		Id:        0,
		NodeId:    errNodeId.Marshal(),
		Error:     "test err",
		Signature: nil,
	}

	loadedKey, err := rsa.LoadPrivateKeyFromPem(nodeKey)
	if err != nil {
		t.Error("Failed to load pk")
	}

	err = signature.SignRsa(errMsg, loadedKey)
	if err != nil {
		t.Error("Failed to sign message")
	}

	msg := &pb.PermissioningPoll{
		Error: errMsg,
	}

	nsm := node.NewStateMap()
	_ = nsm.AddNode(errNodeId, "", "", "", 0)
	n := nsm.GetNode(errNodeId)
	rsm := round.NewStateMap()
	s, _ := rsm.AddRound(id.Round(0), 4, 8, 5*time.Minute, connect.NewCircuit([]*id.ID{errNodeId}))
	_ = n.SetRound(s)

	err = verifyError(msg, n, impl)
	if err != nil {
		t.Error("Failed to verify error")
	}
}
