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
)

var nodeAddr = "0.0.0.0:6900"
var nodeCert []byte
var nodeKey []byte

/*
var testPermissioningKey *rsa.PrivateKey
var testpermissioningCert *x509.Certificate*/

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
		Address:       "0.0.0.0:5900",
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
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
		Address:       "0.0.0.0:5900",
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}
	impl := StartRegistration(testParams)

	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")
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
func TestCompleteRegistration_ErrorPath(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {

		}
	}()
	newImpl := RegistrationImpl{}
	newImpl.permissioningKey, newImpl.permissioningCert = initPermissioningServerKeys()

	nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")
	//Insert a sample regCode
	//err := database.PermissioningDb.InsertNodeRegCode("AAAA")

	strings := make([]string, 0)
	strings = append(strings, "BBBB")
	database.PopulateNodeRegistrationCodes(strings)
	RegistrationCodes = strings
	err := newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert), "BBBB", "0.0.0.0:6900")

	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

	newImpl.Comms.Shutdown()

}
