////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package cmd

import (
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/registration/database"
	"gitlab.com/elixxir/registration/testkeys"
	"io/ioutil"
	"testing"
)

func startPermissioningServer() {
	permKeyBytes, _ := ioutil.ReadFile(testkeys.GetCAKeyPath())
	permCertBytes, _ := ioutil.ReadFile(testkeys.GetCACertPath())

	tmpKey, _ := tls.LoadRSAPrivateKey(string(permKeyBytes))
	permissioningKey = &rsa.PrivateKey{*tmpKey}
	//rmissioningKey.PrivateKey = *tmpKey
	permissioningCert, _ = tls.LoadCertificate(string(permCertBytes))

}


//Error path:
func TestKey_EmptyDataBase(t *testing.T) {
	//Start the registration server
	//Pass along channels?
	newImpl := NewRegistrationImpl()
	/**
	//Note that to find where something is wrong in the set private key, glide up and uncomment this block
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

	//Set the permissioning key for testing
	startPermissioningServer()

	//var c client.ClientComms

	nodeCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	database.PermissioningDb = 	database.NewDatabase("test", "password", "regCodes", "0.0.0.0:6900")

	err := newImpl.RegisterNode([]byte("test"), string(nodeCert), string(nodeCert),
		"AAA", "0.0.0.0:5900")
	if err == nil {
		t.Errorf("Database was empty but allowed a reg code to go through")
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
