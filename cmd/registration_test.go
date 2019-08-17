////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package cmd

import (
	"crypto/x509"
	"fmt"
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
	//initPermissioningServerKeys()

	testParams := Params{
		Address:       permAddr,
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}
	impl := StartRegistration(testParams)

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
	//nodeComm.Disconnect("Permissioning")
	impl.Comms.Shutdown()
}

//Happy Path:  Insert a reg code along with a node
func TestRegCodeExists_InsertNode(t *testing.T) {
	//Iniatialize an implementation and the permissioning server
	//initPermissioningServerKeys()
	newImpl := &RegistrationImpl{}

	newImpl.permissioningKey, newImpl.permissioningCert = initPermissioningServerKeys()

	//Inialiaze the database
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")
	//Insert regcodes into it
	err := database.PermissioningDb.InsertClientRegCode("AAAA", 100)
	if err != nil {
		t.Errorf("Failed to insert client reg code %+v", err)
	}

	//Attempt to register a node
	sig, err := newImpl.RegisterUser("AAAA", string(nodeKey))

	if err != nil {
		t.Errorf("Failed to register a node when it should have worked: %+v", err)
	}

	if sig == nil {
		t.Errorf("Failed to sign public key, recieved %+v as a signature", sig)
	}

}

//Attempt to register a node after the
func TestCompleteRegistration_HappyPath(t *testing.T) {

	testParams := Params{
		Address:       permAddr,
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}
	//Need to set the global ndf...change? add to impl??
	RegParams = testParams

	newImpl := StartRegistration(testParams)
	//connect to the node
	permCert, _ := ioutil.ReadFile(testkeys.GetCACertPath())
	_ = nodeComm.ConnectToRemote(connectionID("Permissioning"), permAddr, permCert)

	//nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")
	//Insert a sample regCode
	//err := database.PermissioningDb.InsertNodeRegCode("AAAA")
	//This is the only difference witht hte test above..
	strings := make([]string, 0)
	strings = append(strings, "BBBB")
	database.PopulateNodeRegistrationCodes(strings)
	RegistrationCodes = strings
	err := newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert), "BBBB", "0.0.0.0:6900")
	//So the impl is not destroyed
	time.Sleep(5 * time.Second)

	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}
	nodeComm.Disconnect("Permissioning")
	newImpl.Comms.Shutdown()

}

func TestTopology_MultiNodes(t *testing.T) {
	testParams := Params{
		Address:       permAddr,
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}
	RegParams = testParams

	newImpl := StartRegistration(testParams)
	permCert, _ := ioutil.ReadFile(testkeys.GetCACertPath())
	fmt.Println("starting 2nd node comm")
	nodeComm2 := node.StartNode("0.0.0.0:6901", node.NewImplementation(), nodeCert, nodeKey)
	fmt.Println("connecting 1st node")
	_ = nodeComm.ConnectToRemote(connectionID("Permissioning"), permAddr, permCert)
	fmt.Println("connecting 2nd node")
	_ = nodeComm2.ConnectToRemote(connectionID("Permissioning"), permAddr, permCert)
	fmt.Println("starting database")
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")
	//Insert a sample regCode
	//err := database.PermissioningDb.InsertNodeRegCode("AAAA")

	//mock node that has a mock download topology function

	//This is the only difference witht hte test above..
	strings := make([]string, 0)
	strings = append(strings, "BBBB", "CCCC")
	database.PopulateNodeRegistrationCodes(strings)
	RegistrationCodes = strings
	fmt.Println("registering 1st node")
	err := newImpl.RegisterNode([]byte("A"), string(nodeCert), string(nodeCert), "BBBB", nodeAddr)

	//newImpl2 := StartRegistration(testParams)
	//fmt.Printf("at initalizationg, conn manager is (newimpl2): %+v\n", newImpl.Comms.ConnectionManager)
	fmt.Printf("Registering 2nd node\n")
	//Does this need to be in the code somewhere, or am I doing a dumb thing?
	err = newImpl.RegisterNode([]byte("B"), string(nodeCert), string(nodeCert), "CCCC", "0.0.0.0:6901")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}
	time.Sleep(5 * time.Second)

	fmt.Println("disconnecting")
	nodeComm.Disconnect("Permissioning")
	nodeComm2.Disconnect("Permissioning")

	newImpl.Comms.Shutdown()
}
