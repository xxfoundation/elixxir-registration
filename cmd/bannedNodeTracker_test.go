////////////////////////////////////////////////////////////////////////////////
// Copyright © 2019 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package cmd

import (
	"crypto/rand"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"testing"
)

func TestBannedNodeTracker(t *testing.T) {
	//Create database
	var err error
	storage.PermissioningDb, err = storage.NewDatabase("test", "password",
		"regCodes", "0.0.0.0", "-1")
	if err != nil {
		t.Errorf("%+v", err)
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Call ban on an empty database
	err = BannedNodeTracker(testState)
	if err != nil {
		t.Errorf("Unexpected error in happy path: %v", err)
	}

	err = storage.PermissioningDb.RegisterNode(id.NewIdFromString("B", id.Node, t),
		nodeAddr, string(nodeCert),
		"0.0.0.0:7900", string(gatewayCert), "BBBB")
	if err != nil {
		t.Errorf("Failed to insert node: %v", err)
	}

	storage.PermissioningDb.set()

}
