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
		Address:"0.0.0.0:5800",
		CertPath:testkeys.GetNodeCertPath(),
		KeyPath:testkeys.GetNodeKeyPath(),
		NdfOutputPath:testkeys.GetNDFPath(),
	}

	StartRegistration(testParams)
}