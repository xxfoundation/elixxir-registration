////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package cmd

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	nodeComms "gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/testkeys"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"sync"
	"testing"
	"time"
)

var nodeAddr = "0.0.0.0:6900"
var nodeCert []byte
var nodeKey []byte
var permAddr = "0.0.0.0:5900"
var testParams Params
var gatewayCert []byte

var nodeComm *nodeComms.Comms

var dblck sync.Mutex

func TestMain(m *testing.M) {
	jww.SetStdoutThreshold(jww.LevelDebug)

	connect.TestingOnlyDisableTLS = true

	var err error
	nodeCert, err = utils.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		fmt.Printf("Could not get node cert: %+v\n", err)
	}

	nodeKey, err = utils.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		fmt.Printf("Could not get node key: %+v\n", err)
	}

	gatewayCert, err = utils.ReadFile(testkeys.GetCACertPath())
	if err != nil {
		fmt.Printf("Could not get gateway cert: %+v\n", err)
	}

	minGatewayVersion, err := version.ParseVersion("1.1.0")
	if err != nil {
		fmt.Printf("Could not parse gateway version: %+v\n", err)
	}

	minServerVersion, err := version.ParseVersion("1.1.0")
	if err != nil {
		fmt.Printf("Could not parse server version: %+v\n", err)
	}

	testParams = Params{
		Address:             permAddr,
		CertPath:            testkeys.GetCACertPath(),
		KeyPath:             testkeys.GetCAKeyPath(),
		FullNdfOutputPath:   testkeys.GetNDFPath(),
		publicAddress:       permAddr,
		udbCertPath:         testkeys.GetUdbCertPath(),
		NsCertPath:          testkeys.GetUdbCertPath(),
		minimumNodes:        3,
		minGatewayVersion:   minGatewayVersion,
		minServerVersion:    minServerVersion,
		disableGeoBinning:   true,
		pruneRetentionLimit: 500 * time.Millisecond,
	}
	nodeComm = nodeComms.StartNode(&id.TempGateway, nodeAddr, 0, nodeComms.NewImplementation(), nodeCert, nodeKey)

	runFunc := func() int {
		code := m.Run()
		nodeComm.Shutdown()
		return code
	}

	os.Exit(runFunc())
}

// Error path: Test an insertion on an empty database
func TestEmptyDataBase(t *testing.T) {

	dblck.Lock()
	defer dblck.Unlock()
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	// Start the registration server
	testParams := Params{
		CertPath:           testkeys.GetCACertPath(),
		KeyPath:            testkeys.GetCAKeyPath(),
		udbCertPath:        testkeys.GetUdbCertPath(),
		NsCertPath:         testkeys.GetUdbCertPath(),
		disableGeoBinning:  true,
		WhitelistedIdsPath: testkeys.GetPreApprovedPath(),
	}
	// Start registration server
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf(err.Error())
	}

	// using node cert as gateway cert
	err = impl.RegisterNode([]byte("test"), nodeAddr, string(nodeCert),
		nodeAddr, string(nodeCert), "AAA")
	if err == nil {
		expectedErr := "Unable to insert node: unable to register node AAA"
		t.Errorf("Database was empty but allowed a reg code to go through. "+
			"Expected %s, Recieved: %+v", expectedErr, err)
		return
	}
}

// Happy path: looking for a code that is in the database
func TestRegCodeExists_InsertRegCode(t *testing.T) {

	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	// Start registration server
	testParams.Address = "0.0.0.0:5901"
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	defer impl.Comms.Shutdown()
	dblck.Lock()
	defer dblck.Unlock()

	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	// Insert a sample regCode
	applicationId := uint64(10)
	newNode := &storage.Node{
		Code:          "AAAA",
		Sequence:      "GB",
		ApplicationId: applicationId,
	}
	newApplication := &storage.Application{Id: applicationId}
	err = storage.PermissioningDb.InsertApplication(newApplication, newNode)
	if err != nil {
		t.Errorf("Failed to insert client reg code %+v", err)
	}
	// Register a node with that regCode
	testSalt := []byte("testtesttesttesttesttesttesttest")
	err = impl.RegisterNode(testSalt, nodeAddr, string(nodeCert),
		nodeAddr, string(nodeCert), newNode.Code)
	if err != nil {
		t.Errorf("Registered a node with a known reg code, but recieved the following error: %+v", err)
	}
}

// Attempt to register a node after the
func TestCompleteRegistration_HappyPath(t *testing.T) {
	// Initialize the database
	var err error
	dblck.Lock()
	defer dblck.Unlock()

	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	// Insert a sample regCode
	infos := []node.Info{
		{RegCode: "BBBB", Order: "US"},
	}

	storage.PopulateNodeRegistrationCodes(infos)
	localParams := testParams
	localParams.minimumNodes = 1
	// Start registration server
	impl, err := StartRegistration(localParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	defer impl.Comms.Shutdown()
	RegParams = testParams

	go func() {
		testSalt := []byte("testtesttesttesttesttesttesttest")
		err = impl.RegisterNode(testSalt, "0.0.0.0:6900", string(nodeCert),
			"0.0.0.0:6900", string(nodeCert), "BBBB")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
			return
		}
	}()

	select {
	case <-time.NewTimer(200 * time.Millisecond).C:
		t.Errorf("Registration failed to complete")
		t.FailNow()
	case <-impl.beginScheduling:
	}
}

// Error path: test that trying to register with the same reg code fails
func TestDoubleRegistration(t *testing.T) {
	// Initialize the database
	var err error
	dblck.Lock()
	defer dblck.Unlock()

	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}
	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	// Create reg codes and populate the database
	infos := []node.Info{
		{RegCode: "AAAA", Order: "CR"},
		{RegCode: "BBBB", Order: "GB"},
		{RegCode: "CCCC", Order: "BF"},
	}
	storage.PopulateNodeRegistrationCodes(infos)
	RegParams = testParams

	// Start registration server
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	defer impl.Comms.Shutdown()

	// Create a second node to register
	nodeComm2 := nodeComms.StartNode(&id.TempGateway, "0.0.0.0:6901", 0, nodeComms.NewImplementation(), nodeCert, nodeKey)
	defer nodeComm2.Shutdown()
	// Register 1st node
	testSalt := []byte("testtesttesttesttesttesttesttest")
	err = impl.RegisterNode(testSalt, nodeAddr, string(nodeCert),
		nodeAddr, string(nodeCert), "BBBB")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	// Register 2nd node
	err = impl.RegisterNode(testSalt, "0.0.0.0:6901", string(nodeCert),
		"0.0.0.0:6901", string(nodeCert), "BBBB")
	if err != nil {
		return
	}

	t.Errorf("Expected happy path, recieved error: %+v", err)
}

// Happy path: attempt to register 2 nodes
func TestTopology_MultiNodes(t *testing.T) {
	// Initialize the database
	var err error
	dblck.Lock()
	defer dblck.Unlock()

	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}
	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	// Create reg codes and populate the database
	infos := []node.Info{
		{RegCode: "AAAA", Order: "CR"},
		{RegCode: "BBBB", Order: "GB"},
		{RegCode: "CCCC", Order: "BF"},
	}

	storage.PopulateNodeRegistrationCodes(infos)

	localParams := testParams
	localParams.minimumNodes = 2

	// Start registration server
	impl, err := StartRegistration(localParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	defer impl.Comms.Shutdown()

	// Create a second node to register
	nodeComm2 := nodeComms.StartNode(&id.TempGateway, "0.0.0.0:6901", 0, nodeComms.NewImplementation(), nodeCert, nodeKey)
	// Kill the connections for the next test
	defer nodeComm2.Shutdown()
	go func() {
		testSalt := []byte("testtesttesttesttesttesttesttest")
		// Register 1st node
		err = impl.RegisterNode(testSalt,
			nodeAddr, string(nodeCert),
			nodeAddr, string(nodeCert), "BBBB")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
		}

		// Register 2nd node
		err = impl.RegisterNode(testSalt,
			"0.0.0.0:6901", string(gatewayCert),
			"0.0.0.0:6901", string(gatewayCert), "CCCC")
		if err != nil {
			t.Errorf("Expected happy path, recieved error: %+v", err)
		}
	}()

	select {
	case <-time.NewTimer(250 * time.Millisecond).C:
		t.Errorf("Registration failed to complete")
		t.FailNow()
	case <-impl.beginScheduling:
	}
}

// Happy path
func TestRegistrationImpl_CheckNodeRegistration(t *testing.T) {
	// Initialize the database
	var err error
	dblck.Lock()
	defer dblck.Unlock()

	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}
	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	// Create reg codes and populate the database
	infos := []node.Info{
		{RegCode: "AAAA", Order: "CR"},
		{RegCode: "BBBB", Order: "GB"},
		{RegCode: "CCCC", Order: "BF"},
	}

	storage.PopulateNodeRegistrationCodes(infos)

	localParams := testParams
	localParams.minimumNodes = 2

	// Start registration server
	impl, err := StartRegistration(localParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	// Kill the connections for the next test
	defer impl.Comms.Shutdown()

	// Craft registered node id
	testNodeID := id.NewIdFromString("A", id.Node, t)

	// Register 1st node
	err = impl.RegisterNode(testNodeID.Marshal(),
		nodeAddr, string(nodeCert),
		nodeAddr, string(nodeCert), "BBBB")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	// Craft message with node id registered
	registrationMessage := &pb.RegisteredNodeCheck{
		ID: testNodeID.Bytes(),
	}

	// Check if node that has been registered is registered
	isRegistered, _ := impl.CheckNodeRegistration(registrationMessage)
	if isRegistered {
		t.Errorf("Registration code should have been registered due to missing salt!")
	}

	// Craft unregistered node id
	badNodeId := id.NewIdFromString("C", id.Node, t)

	// Craft message with unregistered node id
	badRegistrationMessage := &pb.RegisteredNodeCheck{
		ID: badNodeId.Bytes(),
	}

	// Check if node that has NOT been registered isn't registered
	isRegistered, _ = impl.CheckNodeRegistration(badRegistrationMessage)
	if isRegistered {
		t.Errorf("Registration code should not have been registered!")
	}

}

// Error path: Pass nil message
func TestCheckRegistration_NilMsg(t *testing.T) {
	// Initialize the database
	var err error
	dblck.Lock()
	defer dblck.Unlock()

	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}
	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	// Create reg codes and populate the database
	infos := []node.Info{
		{RegCode: "AAAA", Order: "CR"},
		{RegCode: "BBBB", Order: "GB"},
		{RegCode: "CCCC", Order: "BF"},
	}

	storage.PopulateNodeRegistrationCodes(infos)

	localParams := testParams
	localParams.minimumNodes = 2

	// Start registration server
	impl, err := StartRegistration(localParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	// Kill the connections for the next test
	defer impl.Comms.Shutdown()

	// Craft registered node id
	testNodeID := id.NewIdFromString("A", id.Node, t)

	// Register 1st node
	err = impl.RegisterNode(testNodeID.Marshal(),
		nodeAddr, string(nodeCert),
		nodeAddr, string(nodeCert), "BBBB")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	_, err = impl.CheckNodeRegistration(nil)
	if err == nil {
		t.Errorf("Expected error path. Should not be able to pass a nil message")
	}
}

// Error path: Check for invalid ID
func TestCheckRegistration_InvalidID(t *testing.T) {
	// Initialize the database
	var err error
	dblck.Lock()
	defer dblck.Unlock()

	storage.PermissioningDb, _, err = storage.NewDatabase("test",
		"password", "regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}
	err = storage.PermissioningDb.InsertEphemeralLength(
		&storage.EphemeralLength{Length: 8, Timestamp: time.Now()})
	if err != nil {
		t.Errorf("Failed to insert ephemeral length into database: %+v", err)
	}

	// Create reg codes and populate the database
	infos := []node.Info{
		{RegCode: "AAAA", Order: "CR"},
		{RegCode: "BBBB", Order: "GB"},
		{RegCode: "CCCC", Order: "BF"},
	}

	storage.PopulateNodeRegistrationCodes(infos)

	localParams := testParams
	localParams.minimumNodes = 2

	// Start registration server
	impl, err := StartRegistration(localParams)
	if err != nil {
		t.Errorf(err.Error())
		return
	}
	// Kill the connections for the next test
	defer impl.Comms.Shutdown()

	// Craft registered node id
	testNodeID := id.NewIdFromString("A", id.Node, t)

	// Register 1st node
	err = impl.RegisterNode(testNodeID.Marshal(),
		nodeAddr, string(nodeCert),
		nodeAddr, string(nodeCert), "BBBB")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	// Craft message with unregistered node id
	badRegistrationMessage := &pb.RegisteredNodeCheck{
		ID: []byte("invalidId"),
	}

	_, err = impl.CheckNodeRegistration(badRegistrationMessage)
	if err == nil {
		t.Errorf("Expected error path. Should not be able to marshall an invalid ID")
	}
}
