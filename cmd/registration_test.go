////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package cmd

import (
	"crypto/x509"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/registration/database"
	"gitlab.com/elixxir/registration/testkeys"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

var nodeAddr = "0.0.0.0:6900"
var nodeCert []byte
var nodeKey []byte
var permAddr = "0.0.0.0:5900"
var testParams Params

/*
var testPermissioningKey *rsa.PrivateKey
var testpermissioningCert *x509.Certificate*/
var nodeComm *node.NodeComms

func TestMain(m *testing.M) {
	var err error
	nodeCert, err = ioutil.ReadFile(testkeys.GetNodeCertPath())
	if err != nil {
		fmt.Printf("Could not get node cert: %+v\n", err)
	}

	nodeKey, err = ioutil.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		fmt.Printf("Could not get node key: %+v\n", err)
	}

	testParams = Params{
		Address:       permAddr,
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}

	nodeComm = node.StartNode(nodeAddr, node.NewImplementation(), nodeCert, nodeKey)

	runFunc := func() int {
		code := m.Run()
		nodeComm.Shutdown()
		return code
	}

	os.Exit(runFunc())
}

//Helper function that initailizes the permisssioning server's globals
//Todo: throw in the permDB??
func initPermissioningServerKeys() (*rsa.PrivateKey, *x509.Certificate) {
	permKeyBytes, _ := ioutil.ReadFile(testkeys.GetCAKeyPath())
	permCertBytes, _ := ioutil.ReadFile(testkeys.GetCACertPath())

	testPermissioningKey, _ := rsa.LoadPrivateKeyFromPem(permKeyBytes)
	testpermissioningCert, _ := tls.LoadCertificate(string(permCertBytes))
	return testPermissioningKey, testpermissioningCert

}

//Error path: Test an insertion on an empty database
func TestEmptyDataBase(t *testing.T) {
	//Start the registration server
	testParams := Params{
		CertPath: testkeys.GetCACertPath(),
		KeyPath:  testkeys.GetCAKeyPath(),
	}
	impl := StartRegistration(testParams)
	//Set the permissioning keys for testing
	//initPermissioningServerKeys()

	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")

	//using node cert as gateway cert
	err := impl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert),
		"AAA", nodeAddr)
	if err == nil {
		expectedErr := "Unable to insert node: unable to register node AAA"
		t.Errorf("Database was empty but allowed a reg code to go through. "+
			"Expected %s, Recieved: %+v", expectedErr, err)
		return
	}
	impl.Comms.Shutdown()

}

//Happy path: looking for a code that is in the database
func TestRegCodeExists_InsertRegCode(t *testing.T) {

	impl := StartRegistration(testParams)
	impl.completedNodes = make(chan struct{}, 1)
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")
	//Insert a sample regCode
	err := database.PermissioningDb.InsertNodeRegCode("AAAA")
	if err != nil {
		t.Errorf("Failed to insert client reg code %+v", err)
	}
	//Register a node with that regCode
	err = impl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert),
		"AAAA", nodeAddr)
	if err != nil {
		t.Errorf("Registered a node with a known reg code, but recieved the following error: %+v", err)
	}

	//Kill the connections for the next test
	impl.Comms.Shutdown()
}

//Happy Path:  Insert a reg code along with a node
func TestRegCodeExists_RegUser(t *testing.T) {
	//Initialize an implementation and the permissioning server
	impl := &RegistrationImpl{}
	impl.completedNodes = make(chan struct{}, 1)

	jww.SetStdoutThreshold(jww.LevelInfo)

	impl.permissioningKey, impl.permissioningCert = initPermissioningServerKeys()

	//Inialiaze the database
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")
	//Insert regcodes into it
	err := database.PermissioningDb.InsertClientRegCode("AAAA", 100)
	if err != nil {
		t.Errorf("Failed to insert client reg code %+v", err)
	}

	//Attempt to register a user
	sig, err := impl.RegisterUser("AAAA", string(nodeKey))

	if err != nil {
		t.Errorf("Failed to register a node when it should have worked: %+v", err)
	}

	if sig == nil {
		t.Errorf("Failed to sign public key, recieved %+v as a signature", sig)
	}

}

//Attempt to register a node after the
func TestCompleteRegistration_HappyPath(t *testing.T) {
	//Crate database
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")
	//Insert a sample regCode
	strings := make([]string, 0)
	strings = append(strings, "BBBB")
	database.PopulateNodeRegistrationCodes(strings)
	RegistrationCodes = strings

	//Start the registration server
	impl := StartRegistration(testParams)
	RegParams = testParams
	go NodeRegistrationCompleter(impl)

	//connect the node to the permissioning server
	permCert, _ := ioutil.ReadFile(testkeys.GetCACertPath())
	_ = nodeComm.ConnectToRemote(connectionID("Permissioning"), permAddr, permCert, false)

	//nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())

	err := impl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert), "BBBB", "0.0.0.0:6900")
	//So the impl is not destroyed
	time.Sleep(5 * time.Second)

	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}
	//Kill the connections for the next test
	nodeComm.Disconnect("Permissioning")
	impl.Comms.Shutdown()

}

//Error path: test that trying to register with the same reg code fails
func TestDoubleRegistration(t *testing.T) {
	//Create database
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")

	//Create reg codes and populate the database
	strings := make([]string, 0)
	strings = append(strings, "AAAA", "BBBB", "CCCC")
	database.PopulateNodeRegistrationCodes(strings)
	RegistrationCodes = strings
	RegParams = testParams

	//Start registration server
	impl := StartRegistration(testParams)
	go NodeRegistrationCompleter(impl)

	permCert, _ := ioutil.ReadFile(testkeys.GetCACertPath())

	//Create a second node to register
	nodeComm2 := node.StartNode("0.0.0.0:6901", node.NewImplementation(), nodeCert, nodeKey)

	//Connect both nodes to the registration server
	_ = nodeComm.ConnectToRemote(connectionID("Permissioning"), permAddr, permCert, false)
	_ = nodeComm2.ConnectToRemote(connectionID("Permissioning"), permAddr, permCert, false)

	//Register 1st node
	err := impl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert), "BBBB", nodeAddr)
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	//Register 2nd node
	err = impl.RegisterNode([]byte("B"), string(nodeCert), string(nodeCert), "BBBB", "0.0.0.0:6901")
	//Kill the connections for the next test
	nodeComm.Disconnect("Permissioning")
	nodeComm2.Disconnect("Permissioning")
	nodeComm2.Shutdown()
	impl.Comms.Shutdown()
	time.Sleep(5 * time.Second)
	if err != nil {
		return
	}

	t.Errorf("Expected happy path, recieved error: %+v", err)
}

//Happy path: attempt to register 2 nodes
func TestTopology_MultiNodes(t *testing.T) {
	//Create database
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")

	//Create reg codes and populate the database
	strings := make([]string, 0)
	strings = append(strings, "BBBB", "CCCC")
	database.PopulateNodeRegistrationCodes(strings)
	RegistrationCodes = strings
	RegParams = testParams

	//Start registration server
	impl := StartRegistration(testParams)
	go NodeRegistrationCompleter(impl)

	permCert, _ := ioutil.ReadFile(testkeys.GetCACertPath())

	//Create a second node to register
	nodeComm2 := node.StartNode("0.0.0.0:6901", node.NewImplementation(), nodeCert, nodeKey)

	//Connect both nodes to the registration server
	_ = nodeComm.ConnectToRemote(connectionID("Permissioning"), permAddr, permCert, false)
	_ = nodeComm2.ConnectToRemote(connectionID("Permissioning"), permAddr, permCert, false)

	//Register 1st node
	err := impl.RegisterNode([]byte("A"), string(nodeCert), string(nodeCert), "BBBB", nodeAddr)
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	//Register 2nd node
	err = impl.RegisterNode([]byte("B"), string(nodeCert), string(nodeCert), "CCCC", "0.0.0.0:6901")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}
	//Sleep so that the permisioning has time to connect to the nodes (ie impl isn't destroyed)
	time.Sleep(5 * time.Second)

	//Kill the connections for the next test
	nodeComm.Disconnect("Permissioning")
	nodeComm2.Disconnect("Permissioning")
	nodeComm2.Shutdown()
	impl.Comms.Shutdown()
}
