////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"crypto/rand"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/region"
	"testing"
)

func TestBannedNodeTracker(t *testing.T) {
	//Create database
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("",
		"", "", "", "")
	if err != nil {
		t.Errorf("%+v", err)
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", "", region.GetCountryBins())
	impl := &RegistrationImpl{
		State: testState,
	}
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Call ban on an empty database
	err = BannedNodeTracker(impl)
	if err != nil {
		t.Errorf("Unexpected error in happy path: %v", err)
	}

	// Create an active and banned node
	bannedNode := createNode(testState, "0", "AAA", 10, node.Banned, t)
	activeNode := createNode(testState, "1", "BBB", 20, node.Active, t)
	curDef := testState.GetUnprunedNdf()
	curDef.Nodes = append(curDef.Nodes, ndf.Node{
		ID:             bannedNode.Marshal(),
		Address:        "",
		TlsCertificate: "",
	})
	curDef.Nodes = append(curDef.Nodes, ndf.Node{
		ID:             activeNode.Marshal(),
		Address:        "",
		TlsCertificate: "",
	})
	testState.UpdateInternalNdf(curDef)
	err = testState.UpdateOutputNdf()
	if err != nil {
		t.Error("Failed to output test state ndf")
	}

	// Clean out banned nodes
	err = BannedNodeTracker(impl)
	if err != nil {
		t.Errorf("Error with node tracker: %v", err)
	}

	updatedDef := testState.GetUnprunedNdf()
	if len(updatedDef.Nodes) != 1 {
		t.Error("Banned node tracker did not alter ndf")
	}

	// Check that the banned node has been updated to banned
	receivedBannedNode := testState.GetNodeMap().GetNode(bannedNode)
	if !receivedBannedNode.IsBanned() {
		t.Errorf("Node expected to be banned: %v", receivedBannedNode.GetStatus())
	}

	// Check that the allowed node has not been updated to banned
	receivedAllowedNode := testState.GetNodeMap().GetNode(activeNode)
	if receivedAllowedNode.IsBanned() {
		t.Errorf("Node expected to be banned: %v", receivedAllowedNode.GetStatus())
	}

	// Clean out banned nodes again. Check that it does not attempt to
	// ban an already banned node
	err = BannedNodeTracker(impl)
	if err != nil {
		t.Errorf("Error with node tracker: %v", err)
	}
}

func createNode(testState *storage.NetworkState, order, regCode string, appId int,
	status node.Status, t *testing.T) *id.ID {
	// Create new byte slice of the correct size
	idBytes := make([]byte, id.ArrIDLen)

	// Create random bytes
	_, err := rand.Read(idBytes)
	if err != nil {
		t.Fatalf("Failed to generate random bytes: %v", err)
	}

	// Create a node with a banned status
	applicationId := uint64(appId)
	testNode := &storage.Node{
		Id:            idBytes,
		Code:          regCode,
		Sequence:      order,
		ApplicationId: applicationId,
		Status:        uint8(status),
	}

	newApplication := &storage.Application{Id: applicationId}

	// Insert banned node into database
	err = storage.PermissioningDb.InsertApplication(newApplication, testNode)
	if err != nil {
		t.Errorf("Failed to insert client reg code %+v", err)
	}

	nodeId := id.NewIdFromBytes(idBytes, t)

	// Add node to the mao
	err = testState.GetNodeMap().AddNode(nodeId, order, "", "", 1)
	if err != nil {
		t.Errorf("Failed to add node to node map: %v", err)
	}

	return nodeId
}
