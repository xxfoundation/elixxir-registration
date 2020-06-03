////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"bytes"
	"fmt"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/testkeys"
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
func TestRegistrationImpl_Poll(t *testing.T) {
	testID := id.NewIdFromUInt(0, id.Node, t)
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
		Gateways: []ndf.Gateway{
			{ID: id.NewIdFromUInt(0, id.Gateway, t).Bytes()},
		},
		Nodes: []ndf.Node{
			{ID: id.NewIdFromUInt(0, id.Node, t).Bytes()},
		},
	})

	// Make a simple auth object that will pass the checks
	testHost, _ := connect.NewHost(testID, testString,
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
		LastUpdate:     0,
		Activity:       uint32(current.WAITING),
		Error:          nil,
		GatewayVersion: "1.1.0",
		ServerVersion:  "1.1.0",
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

	response, err := impl.Poll(testMsg, testAuth, "0.0.0.0")
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
	testVersion, _ := version.ParseVersion("0.0.0")
	impl := &RegistrationImpl{
		State:    state,
		NdfReady: &ndfReady,
		params: &Params{
			minGatewayVersion: testVersion,
			minServerVersion:  testVersion,
		},
	}

	_, err = impl.Poll(nil, nil, "")
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
	testVersion, _ := version.ParseVersion("0.0.0")
	impl := RegistrationImpl{
		State:    state,
		NdfReady: &ndfReady,
		params: &Params{
			minGatewayVersion: testVersion,
			minServerVersion:  testVersion,
		},
	}

	err = impl.State.UpdateNdf(&ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:        "420",
			TlsCertificate: "",
		},
	})

	// Make a simple auth object that will fail the checks
	testHost, _ := connect.NewHost(id.NewIdFromString(testString, id.Node, t),
		testString, make([]byte, 0), false, true)
	testAuth := &connect.Auth{
		IsAuthenticated: false,
		Sender:          testHost,
	}

	_, err = impl.Poll(nil, testAuth, "0.0.0.0")
	if err == nil || err.Error() != connect.AuthError(testAuth.Sender.GetId()).Error() {
		t.Errorf("Unexpected error polling: %+v", err)
	}
}

//Happy path
func TestRegistrationImpl_PollNdf(t *testing.T) {
	//Create database
	var err error
	storage.PermissioningDb, err = storage.NewDatabase("test", "password",
		"regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	//Create reg codes and populate the database
	infos := make([]node.Info, 0)
	infos = append(infos, node.Info{RegCode: "BBBB", Order: "0"},
		node.Info{RegCode: "CCCC", Order: "1"},
		node.Info{RegCode: "DDDD", Order: "2"})
	storage.PopulateNodeRegistrationCodes(infos)

	RegParams = testParams
	udbId := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	RegParams.udbId = udbId
	RegParams.minimumNodes = 3
	fmt.Println("-A")
	// Start registration server
	impl, err := StartRegistration(RegParams)
	if err != nil {
		t.Errorf(err.Error())
	}

	var l sync.Mutex
	go func() {
		l.Lock()
		defer l.Unlock()
		fmt.Println("A")
		//Register 1st node
		err = impl.RegisterNode(id.NewIdFromString("B", id.Node, t),
			nodeAddr, string(nodeCert),
			"0.0.0.0:7900", string(gatewayCert), "BBBB")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
		}
		fmt.Println("B")
		//Register 2nd node
		err = impl.RegisterNode(id.NewIdFromString("C", id.Node, t),
			"0.0.0.0:6901", string(nodeCert),
			"0.0.0.0:7901", string(gatewayCert), "CCCC")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
		}
		fmt.Println("C")
		//Register 3rd node
		err = impl.RegisterNode(id.NewIdFromString("D", id.Node, t),
			"0.0.0.0:6902", string(nodeCert),
			"0.0.0.0:7902", string(gatewayCert), "DDDD")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
		}
	}()

	expectedNodeIDs := []*id.ID{id.NewIdFromString("B", id.Node, t),
		id.NewIdFromString("C", id.Node, t), id.NewIdFromString("D", id.Node, t)}

	//wait for registration to complete
	select {
	case <-time.NewTimer(1000 * time.Millisecond).C:
		t.Errorf("Node registration never completed")
		t.FailNow()
	case <-impl.beginScheduling:
	}

	l.Lock()
	observedNDFBytes, err := impl.PollNdf(nil, &connect.Auth{})
	l.Unlock()
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

	for i := range observedNDF.Nodes {
		if bytes.Compare(expectedNodeIDs[i].Bytes(),
			observedNDF.Nodes[i].ID) != 0 {
			t.Errorf("Could not build node %d's id id: Expected: %v \nRecieved: %v", i,
				expectedNodeIDs[i].String(), observedNDF.Nodes[i].ID)
		}
	}

	//Shutdown node comms
	impl.Comms.Shutdown()
}

//Error  path
func TestRegistrationImpl_PollNdf_NoNDF(t *testing.T) {
	//Create database
	var err error
	storage.PermissioningDb, err = storage.NewDatabase("test", "password",
		"regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	//Create reg codes and populate the database
	infos := make([]node.Info, 0)
	infos = append(infos, node.Info{RegCode: "BBBB", Order: "0"},
		node.Info{RegCode: "CCCC", Order: "1"},
		node.Info{RegCode: "DDDD", Order: "2"})
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

	//Register 1st node
	err = impl.RegisterNode(id.NewIdFromString("B", id.Node, t), nodeAddr, string(nodeCert),
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

func TestPoll_BannedNode(t *testing.T) {
	//Create database
	var err error
	storage.PermissioningDb, err = storage.NewDatabase("test", "password",
		"regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	testID := id.NewIdFromUInt(0, id.Node, t)
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
	testHost, _ := connect.NewHost(testID, testString,
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

	err = impl.State.GetNodeMap().AddNode(testID, "", "", "")
	if err != nil {
		t.Errorf("Could nto add node: %s", err)
	}

	impl.State.GetNodeMap().GetNode(testID).Ban()

	_, err = impl.Poll(testMsg, testAuth, "")
	if err != nil {
		return
	}

	t.Errorf("Expected error state: Node with out of network status should return an error")
}

// Check that checkVersion() correctly determines the message versions to be
// compatible with the required versions.
func TestCheckVersion(t *testing.T) {
	testMsg := &pb.PermissioningPoll{
		ServerVersion:  "1.5.6",
		GatewayVersion: "1.4.5",
	}

	requiredServer, _ := version.ParseVersion("1.3.2")
	requiredGateway, _ := version.ParseVersion("1.3.2")

	err := checkVersion(requiredGateway, requiredServer, testMsg)
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

	err := checkVersion(requiredGateway, requiredServer, testMsg)
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

	err := checkVersion(requiredGateway, requiredServer, testMsg)
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

	err := checkVersion(requiredGateway, requiredServer, testMsg)
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

	err := checkVersion(requiredGateway, requiredServer, testMsg)
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

	err := checkVersion(requiredGateway, requiredServer, testMsg)
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

	err := checkVersion(requiredGateway, requiredServer, testMsg)
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

	err := checkVersion(requiredGateway, requiredServer, testMsg)
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

	err = impl.State.UpdateNdf(&ndf.NetworkDefinition{
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
