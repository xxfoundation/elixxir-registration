////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package cmd

import (
	"fmt"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/registration/database"
	"gitlab.com/elixxir/registration/testkeys"
	"io/ioutil"
	"testing"
)

//Helper function that initailizes the permisssioning server's globals
//Todo: throw in the permDB??
func startPermissioningServer() {
	permKeyBytes, _ := ioutil.ReadFile(testkeys.GetCAKeyPath())
	permCertBytes, _ := ioutil.ReadFile(testkeys.GetCACertPath())

	tmpKey, _ := tls.LoadRSAPrivateKey(string(permKeyBytes))
	permissioningKey = &rsa.PrivateKey{*tmpKey}
	//rmissioningKey.PrivateKey = *tmpKey
	permissioningCert, _ = tls.LoadCertificate(string(permCertBytes))

}

//Error path: Test an insertion on an empty database
func TestEmptyDataBase(t *testing.T) {
	//Start the registration server
	//Pass along channels?
	newImpl := NewRegistrationImpl()
	/**
	//Note that to find where something is wrong in the setprivatekey, glide up and uncomment this block
	testParams := Params{
		Address:       "0.0.0.0:5900",
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}
	//thow in waitgroup, listen for outputs??
	//need this with startPermissioningServer
	go StartRegistration(testParams)
	/**/

	//Set the permissioning key for testing
	startPermissioningServer()

	//var c client.ClientComms

	nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")

	err := newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert),
		"AAA", "0.0.0.0:6900")
	if err == nil {
		expectedErr := "Unable to insert node: unable to register node AAA"
		t.Errorf("Database was empty but allowed a reg code to go through. "+
			"Expected %s, Recieved: %+v", expectedErr, err)
		return
	}

	/**
		serverCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
		gateway, _ := ioutil.ReadFile(testkeys.GetNodeKeyPath())
		newImpl.RegisterUser()
		newImpl.RegisterNode(connectionID(0), string(serverCert), )

	/* */
}

//func TestKey_IncorrectRegCode

//Testing: create a reg server that has some code

//Happy path: looking for a code that is in the database
func TestRegCodeExists_InsertRegCode(t *testing.T) {
	newImpl := NewRegistrationImpl()
	startPermissioningServer()

	nodeCert, _ := ioutil.ReadFile((testkeys.GetNodeCertPath()))
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")
	//Insert a sample regCode
	err := database.PermissioningDb.InsertNodeRegCode("AAAA")
	if err != nil {
		t.Errorf("Failed to insert client reg code %+v", err)
	}
	//Register a node with that regCode
	err = newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert),
		"AAAA", "0.0.0.0:6900")

	if err != nil {
		t.Errorf("Registered a node with a known reg code, but recieved the following error: %+v", err)
	}
}

//Happy Path:  Insert a reg code along with a node
func TestRegCodeExists_InsertUser(t *testing.T) {
	//Iniatialize an implementation and the permissioning server
	newImpl := NewRegistrationImpl()
	startPermissioningServer()

	//Inialiaze the database
	nodeKey, _ := ioutil.ReadFile((testkeys.GetClientPublicKey()))
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")
	//Insert regcodes into it
	err := database.PermissioningDb.InsertClientRegCode("AAAA", 100)
	if err != nil {
		t.Errorf("Failed to insert client reg code %+v", err)
	}

	//Attempt to register a node
	_, err = newImpl.RegisterUser("AAAA", string(nodeKey))

	if err != nil {
		t.Errorf("Failed to register a node when it should have worked: %+v", err)
	}

}

//Attempt to register a node after the
func TestCompleteRegistration_ErrorPath(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {

		}
	}()
	newImpl := NewRegistrationImpl()
	/**/
	//Note that to find where something is wrong in the setprivatekey, glide up and uncomment this block
	testParams := Params{
		Address:       "0.0.0.0:5900",
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}
	//thow in waitgroup, listen for outputs??
	//need this with startPermissioningServer
	//go StartRegistration(testParams)
	/**/
	RegParams = testParams
	startPermissioningServer()

	nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")
	//Insert a sample regCode
	//err := database.PermissioningDb.InsertNodeRegCode("AAAA")

	strings := make([]string, 0)
	fmt.Println(strings)
	strings = append(strings, "BBBB")
	database.PopulateNodeRegistrationCodes(strings)
	RegistrationCodes = strings
	err := newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert), "BBBB", "0.0.0.0:6900")

	if err != nil {
		t.Errorf("Expected happy path, recieved error: %+v", err)
	}

}

/**
//Attempt to register a node after the
func TestCompleteRegistration_ErrorPath(t *testing.T)  {
	defer func() {
		if r := recover(); r != nil {

		}
	}()
	newImpl := NewRegistrationImpl()
	/**
	//Note that to find where something is wrong in the setprivatekey, glide up and uncomment this block
	testParams := Params{
		Address:       "0.0.0.0:5900",
		CertPath:      testkeys.GetCACertPath(),
		KeyPath:       testkeys.GetCAKeyPath(),
		NdfOutputPath: testkeys.GetNDFPath(),
	}
	//thow in waitgroup, listen for outputs??
	//need this with startPermissioningServer
	go StartRegistration(testParams)
	/**
	RegParams = testParams
	startPermissioningServer()

	nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	database.PermissioningDb = database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")
	//Insert a sample regCode
	//err := database.PermissioningDb.InsertNodeRegCode("AAAA")

	strings := make([]string, 0)
	fmt.Println(strings)
	strings = append(strings,"BBBB")
	database.PopulateNodeRegistrationCodes(strings)
	RegistrationCodes = strings
	err := newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert), "BBBB", "0.0.0.0:6900")
	fmt.Println(err)
	err = newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert), "BBBB", "0.0.0.0:6900")
	fmt.Println(err)

}*/
