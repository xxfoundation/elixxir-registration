////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"errors"
	"github.com/jinzhu/gorm"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/region"
	"testing"
)

// Happy path
func TestDatabaseImpl_InsertApplication(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_InsertApplication", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	// Attempt to load in a valid code
	applicationId := uint64(10)
	newNode := &Node{
		Code:          "TEST",
		Sequence:      region.NorthAmerica.String(),
		ApplicationId: applicationId,
	}
	newApplication := &Application{Id: applicationId}
	err = d.InsertApplication(newApplication, newNode)
	// Verify the insert was successful
	if err != nil {
		t.Errorf("Expected to successfully insert node registration code")
	}

	res, err := d.GetNode(newNode.Code)
	if err != nil || res.Code != newNode.Code {
		t.Fatalf("Expected insert to add node: %+v", err)
	}

	if res.Sequence != newNode.Sequence {
		t.Errorf("Order string incorret; Expected: %s, Recieved: %s",
			newNode.Sequence, res.Sequence)
	}
}

// Error Path: Duplicate node registration code and application
func TestDatabaseImpl_InsertApplication_Duplicate(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_InsertApplication_Duplicate", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	db := d.GetDatabaseImpl(t)

	// Load in a registration code
	applicationId := uint64(10)
	newNode := &Node{
		Code:          "TEST",
		Sequence:      region.MiddleEast.String(),
		ApplicationId: applicationId,
	}
	newApplication := &Application{Id: applicationId}

	// Attempt to load in a duplicate application

	err = db.db.Create(newApplication).Error
	if err != nil {
		t.Fatalf("Failed to create new application: %+v", err)
	}
	err = d.InsertApplication(newApplication, newNode)
	// Verify the insert failed
	if err == nil {
		t.Errorf("Expected to fail inserting duplicate application")
	}
	err = db.db.Delete(newApplication).Error
	if err != nil {
		t.Fatalf("Failed to delete application for duplicate test: %+v", err)
	}

	// Attempt to load in a duplicate code
	// FIXME this property is not enforced on the database...
	altAppId := uint64(20)
	err = d.InsertApplication(&Application{Id: altAppId}, &Node{
		Code:          "TEST",
		Sequence:      region.EasternEurope.String(),
		ApplicationId: altAppId,
	})
	if err != nil {
		t.Fatalf("Failed to add new node: %+v", err)
	}

	err = d.InsertApplication(newApplication, newNode)
	if err != nil {
		t.Errorf("Second insert will not succeed, but shoudl not throw an error: %+v", err)
	}

	var res []Node
	// Verify the insert failed
	err = db.db.Find(&res, "code = ?", "TEST").Error
	if err != nil {
		t.Fatal(err)
	}

	if len(res) > 1 {
		t.Fatalf("Should not be able to have two nodes on same reg code")
	}

	received := res[0]
	if received.Sequence == region.EasternEurope.String() {
		t.Fatalf("Got wrong region")
	}

	err = db.db.Delete(newNode).Error
	if err != nil {
		t.Fatalf("Failed to delete node for duplicate test")
	}
}

// Happy path
func TestDatabaseImpl_RegisterNode(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_RegisterNode", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	// Load in a registration code
	code := "TEST"
	cert := "cert"
	gwCert := "gwcert"
	addr := "addr"
	gwAddr := "gwaddr"
	applicationId := uint64(10)
	err = d.InsertApplication(&Application{Id: applicationId}, &Node{Code: code})
	if err != nil {
		t.Fatalf("Failed to set up reg code for registernode test: %+v", err)
	}

	// Attempt to insert a node
	err = d.RegisterNode(id.NewIdFromString("", id.Node, t), []byte("test"), code, addr,
		cert, gwAddr, gwCert)
	if err != nil {
		t.Fatalf("Failed call to RegisterNode: %+v", err)
	}

	// Verify the insert was successful
	if info, err := d.GetNode(code); err != nil || info.NodeCertificate != cert ||
		info.GatewayCertificate != gwCert || info.ServerAddress != addr ||
		info.GatewayAddress != gwAddr {
		t.Errorf("Expected to successfully insert node information: %+v", info)
	}
}

// Error path: Invalid registration code
func TestDatabaseImpl_RegisterNode_Invalid(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_RegisterNode_Invalid", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	// Do NOT load in a registration code
	code := "TEST"

	// Attempt to insert a node without an associated registration code
	err = d.RegisterNode(id.NewIdFromString("", id.Node, t), []byte("test"), code, code,
		code, code, code)
	// Verify the insert failed
	// TODO this does not error in sqlite; update not finding rows is not an error in either sql implementation, but psql WILL error with foreign key issues
	if err != nil {
		t.Errorf("This will not return an error for lack of rows: %+v", err)
	}

	_, err = d.GetNode(code)
	if err == nil {
		t.Fatalf("Expected error getting node")
	}
}

// Happy path
func TestDatabaseImpl_GetNode(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetNode", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	applicationId := uint64(10)
	code := "TEST"

	err = d.InsertApplication(&Application{Id: applicationId}, &Node{Code: code})
	if err != nil {
		t.Fatalf("Failed to set up reg code for registernode test: %+v", err)
	}

	// Check that the correct node is obtained
	info, err := d.GetNode(code)
	if err != nil || info.Code != code {
		t.Errorf("Expected to be able to obtain correct node")
	}
}

// Error path: Nonexistent registration code
func TestDatabaseImpl_GetNode_Invalid(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetNode_Invalid", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	// Check that no node is obtained from empty map
	_, err = d.GetNode("TEST")
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("Expected to not find the node")
	}
}

// Happy path
func TestDatabaseImpl_GetNodeById(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetNodeById", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	// Load in a registration code
	code := "TEST"
	testId := id.NewIdFromString(code, id.Node, t)
	applicationId := uint64(10)
	err = d.InsertApplication(&Application{Id: applicationId}, &Node{Code: code, Id: testId.Marshal()})
	if err != nil {
		t.Fatalf("Failed to set up reg code for registernode test: %+v", err)
	}

	// Check that the correct node is obtained
	info, err := d.GetNodeById(testId)
	if err != nil || info.Code != code {
		t.Errorf("Expected to be able to obtain correct node")
	}
}

// Error path: Nonexistent node id
func TestDatabaseImpl_GetNodeById_Invalid(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetNodeById_Invalid", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	testId := id.NewIdFromString("test", id.Node, t)

	// Check that no node is obtained from empty map
	_, err = d.GetNodeById(testId)
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("Expected to not find the node")
	}
}

// Happy path
func TestDatabaseImpl_GetNodesByStatus(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetNodesByStatus", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	db := d.GetDatabaseImpl(t)

	// Should start off empty
	nodes, err := d.GetNodesByStatus(node.Banned)
	if err != nil {
		t.Errorf("Unable to get nodes by status: %+v", err)
	}
	if len(nodes) > 0 {
		t.Errorf("Unexpected nodes returned for status: %v", nodes)
	}

	// Load in a registration code
	code := "TEST"
	testId := id.NewIdFromString(code, id.Node, t)
	applicationId := uint64(10)
	err = d.InsertApplication(&Application{Id: applicationId}, &Node{Code: code, Id: testId.Marshal(), Status: uint8(node.Banned)})
	if err != nil {
		t.Fatalf("Failed to set up reg code for registernode test: %+v", err)
	}

	// Should have a result now
	nodes, err = d.GetNodesByStatus(node.Banned)
	if err != nil {
		t.Errorf("Unable to get nodes by status: %+v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("Unexpected nodes returned for status: %v", nodes)
	}

	// Unban the node
	err = db.db.Model(&Node{}).Where("code = ?", code).Update("status", uint8(node.Active)).Error
	if err != nil {
		t.Fatalf("Failed to unban node in db: %+v", err)
	}

	// Shouldn't get a result anymore
	nodes, err = d.GetNodesByStatus(node.Banned)
	if err != nil {
		t.Errorf("Unable to get nodes by status: %+v", err)
	}
	if len(nodes) > 0 {
		t.Errorf("Unexpected nodes returned for status: %v", nodes)
	}
}

// Happy path
func TestDatabaseImpl_UpdateNodeAddresses(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_UpdateNodeAddresses", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	testString := "test"
	testId := id.NewIdFromString(testString, id.Node, t)
	testResult := "newAddr"
	applicationId := uint64(10)
	err = d.InsertApplication(&Application{Id: applicationId}, &Node{
		Code:           testString,
		Id:             testId.Marshal(),
		ServerAddress:  testString,
		GatewayAddress: testString,
		ApplicationId:  applicationId,
	})
	if err != nil {
		t.Fatalf("Failed to insert data for updateAddress test")
	}

	err = d.UpdateNodeAddresses(testId, testResult, testResult)
	if err != nil {
		t.Errorf(err.Error())
	}

	result, err := d.GetNode(testString)
	if err != nil {
		t.Fatalf("Failed to get node: %+v", err)
	}
	if result.ServerAddress != testResult || result.GatewayAddress != testResult {
		t.Errorf("Field values did not update correctly, got Node %s Gateway %s",
			result.ServerAddress, result.GatewayAddress)
	}
}

// Happy path
func TestDatabaseImpl_UpdateSequence(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_UpdateSequence", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	testString := region.NorthAmerica.String()
	testId := id.NewIdFromString(testString, id.Node, t)
	testResult := "newAddr"
	applicationId := uint64(10)
	err = d.InsertApplication(&Application{Id: applicationId}, &Node{
		Code:           testString,
		Id:             testId.Marshal(),
		Sequence:       testString,
		ServerAddress:  testString,
		GatewayAddress: testString,
		ApplicationId:  applicationId,
	})
	if err != nil {
		t.Fatalf("Failed to insert data for updateAddress test")
	}

	err = d.UpdateNodeSequence(testId, testResult)
	if err != nil {
		t.Errorf(err.Error())
	}

	result, err := d.GetNode(testString)
	if err != nil {
		t.Fatalf("Failed to get node: %+v", err)
	}
	if result.Sequence != testResult {
		t.Errorf("Sequence values did not update correctly, got %s expected %s",
			result.Sequence, testResult)
	}
}
