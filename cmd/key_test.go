////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package cmd

import (
	"encoding/pem"
	"fmt"
	"gitlab.com/elixxir/registration/testkeys"
	"io/ioutil"
	"testing"
)

func TestKey(t *testing.T){
	privKeyBytes, err := ioutil.ReadFile(testkeys.GetNodeKeyPath())
	if err != nil {
		t.Error("Failed to open file in testKeys")
	}

	keyDecode, _ := pem.Decode(privKeyBytes)
	fmt.Println(keyDecode)
	newImpl := NewRegistrationImpl()
	fmt.Println(newImpl)
	testParams := Params{
		Address:"0.0.0.0:5900",
		CertPath:testkeys.GetCACertPath(),
		KeyPath:testkeys.GetCAKeyPath(),
		NdfOutputPath:testkeys.GetNDFPath(),
	}
	//thow in waitgroup, listen for outputs
	StartRegistration(testParams)
/*
	serverCert, _ := ioutil.ReadFile(testkeys.GetNodeCertPath())
	gateway, _ := ioutil.ReadFile(testkeys.GetNodeKeyPath())
	newImpl.RegisterUser()
	newImpl.RegisterNode(connectionID(0), string(serverCert), )*/
}