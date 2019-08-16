////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package cmd

import (
	"fmt"
	"gitlab.com/elixxir/comms/node"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/registration/database"
	"gitlab.com/elixxir/registration/testkeys"
	"io/ioutil"
	"os"
	"testing"
)

var nodeAddr = "0.0.0.0:6900"
var nodeCert []byte
var nodeKey []byte

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

	nodeComm := node.StartNode(nodeAddr, node.NewImplementation(), nodeCert, nodeKey)

	runFunc := func() int {
		code := m.Run()
		nodeComm.Shutdown()
		return code
	}
	os.Exit(runFunc())
}

//Helper function that initailizes the permisssioning server's globals
//Todo: throw in the permDB??
func initPermissioningServerKeys() {
	permKeyBytes, _ := ioutil.ReadFile(testkeys.GetCAKeyPath())
	permCertBytes, _ := ioutil.ReadFile(testkeys.GetCACertPath())

	permissioningKey, _ = rsa.LoadPrivateKeyFromPem(permKeyBytes)
	permissioningCert, _ = tls.LoadCertificate(string(permCertBytes))

}

//Error path: Test an insertion on an empty database
func TestEmptyDataBase(t *testing.T) {
	//Start the registration server
	testParams := Params{
		Address:       "0.0.0.0:5900",
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}
	impl := StartRegistration(testParams)

	//Set the permissioning keys for testing
	initPermissioningServerKeys()

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
	initPermissioningServerKeys()

	testParams := Params{
		Address:       "0.0.0.0:5900",
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
		"AAAA", "0.0.0.0:6900")

	if err != nil {
		t.Errorf("Registered a node with a known reg code, but recieved the following error: %+v", err)
	}

	impl.Comms.Shutdown()
}

//Happy Path:  Insert a reg code along with a node
func TestRegCodeExists_InsertNode(t *testing.T) {
	//Iniatialize an implementation and the permissioning server
	initPermissioningServerKeys()
	newImpl := &RegistrationImpl{}

	//Inialiaze the database
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")
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

//Happy path Attempt to register a node after inserting using PopulateNodeRegistrationCodes
func TestCompleteRegistration_HappyPath(t *testing.T) {
	//This was making it not hit the connect, we will need a mock node
	/*defer func()
		if r := recover(); r != nil {
		}
	}()*/
	//With this, comms is nil and thus returns a seg fault/nil pointer deref
	testParams := Params{
		Address:       "0.0.0.0:5900",
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}

	newImpl := StartRegistration(testParams)
	fmt.Println(newImpl)

	RegParams = testParams
	initPermissioningServerKeys()

	nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6969")
	//Insert a sample regCode
	//err := database.PermissioningDb.InsertNodeRegCode("AAAA")
	fmt.Println("pre popuplate")
	strings := make([]string, 0)
	strings = append(strings, "BBBB")
	database.PopulateNodeRegistrationCodes(strings)
	fmt.Println("post populate")
	RegistrationCodes = strings
	//Attempt to regeist a node with an existing regCode
	err := newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert), "BBBB", nodeAddr)
	fmt.Println("register node post")
	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}
	newImpl.Comms.Shutdown()

}


func TestTopology(t *testing.T)  {
	testParams := Params{
		Address:       "0.0.0.0:5900",
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}

	newImpl := StartRegistration(testParams)

	RegParams = testParams
	initPermissioningServerKeys()

	nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	database.PermissioningDb =  database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")

	regCodes := make([]string,0)
	//Allow for more than one node so they can communicate the topoplogy
	regCodes = append(regCodes,"BBBB", "CCCC")
	database.PopulateNodeRegistrationCodes(regCodes)

	RegistrationCodes = regCodes
	//Should hit complete registration
	_ = newImpl.RegisterNode([]byte("A"), string(nodeCert), string(nodeCert), "BBBB", "0.0.0.0:7900")
	_ = newImpl.RegisterNode([]byte("B"), string(nodeCert), string(nodeCert), "CCCC", "0.0.0.0:8900")

	newImpl.Comms.Shutdown()
}
