////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"bytes"
	"crypto/rand"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Happy path
func TestMapImpl_InsertApplication(t *testing.T) {
	m := &MapImpl{
		nodes:        make(map[string]*Node),
		applications: make(map[uint64]*Application),
	}

	// Attempt to load in a valid code
	applicationId := uint64(10)
	newNode := &Node{
		Code:          "TEST",
		Sequence:      "BLARG",
		ApplicationId: applicationId,
	}
	newApplication := &Application{Id: applicationId}
	err := m.InsertApplication(newApplication, newNode)

	// Verify the insert was successful
	if err != nil || m.nodes[newNode.Code] == nil {
		t.Errorf("Expected to successfully insert node registration code")
	}

	if m.nodes[newNode.Code].Sequence != newNode.Sequence {
		t.Errorf("Order string incorret; Expected: %s, Recieved: %s",
			newNode.Sequence, m.nodes[newNode.Code].Sequence)
	}
}

// Error Path: Duplicate node registration code and application
func TestMapImpl_InsertApplication_Duplicate(t *testing.T) {
	m := &MapImpl{
		nodes:        make(map[string]*Node),
		applications: make(map[uint64]*Application),
	}

	// Load in a registration code
	applicationId := uint64(10)
	newNode := &Node{
		Code:          "TEST",
		Sequence:      "BLARG",
		ApplicationId: applicationId,
	}
	newApplication := &Application{Id: applicationId}

	// Attempt to load in a duplicate application
	m.applications[applicationId] = newApplication
	err := m.InsertApplication(newApplication, newNode)

	// Verify the insert failed
	if err == nil {
		t.Errorf("Expected to fail inserting duplicate application")
	}

	// Attempt to load in a duplicate code
	m.nodes[newNode.Code] = newNode
	err = m.InsertApplication(newApplication, newNode)

	// Verify the insert failed
	if err == nil {
		t.Errorf("Expected to fail inserting duplicate node registration code")
	}
}

// Happy path
func TestMapImpl_UpdateSalt(t *testing.T) {
	testID := id.NewIdFromString("test", id.Node, t)
	key := "testKey"
	newSalt := make([]byte, 8)
	_, _ = rand.Read(newSalt)

	m := &MapImpl{
		nodes: map[string]*Node{key: {Id: testID.Bytes(), Salt: []byte("b")}},
	}

	err := m.UpdateSalt(testID, newSalt)
	if err != nil {
		t.Errorf("Received unexpected error when upadting salt."+
			"\n\terror: %v", err)
	}

	// Verify that the new salt matches the passed in salt
	if !bytes.Equal(newSalt, m.nodes[key].Salt) {
		t.Errorf("Node in map has unexpected salt."+
			"\n\texpected: %d\n\treceived: %d", newSalt, m.nodes[key].Salt)
	}
}

// Tests that MapImpl.UpdateSalt returns an error if no Node is found in the map
// for the given ID.
func TestMapImpl_UpdateSalt_NodeNotInMap(t *testing.T) {
	testID := id.NewIdFromString("test", id.Node, t)
	key := "testKey"
	newSalt := make([]byte, 8)
	_, _ = rand.Read(newSalt)

	m := &MapImpl{
		nodes: map[string]*Node{key: {Id: id.NewIdFromString("test3", id.Node, t).Bytes(), Salt: []byte("b")}},
	}

	err := m.UpdateSalt(testID, newSalt)
	if err == nil {
		t.Errorf("Did not receive an error when the Node does not exist in " +
			"the map.")
	}
}

// Happy path
func TestMapImpl_RegisterNode(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	// Load in a registration code
	code := "TEST"
	cert := "cert"
	gwCert := "gwcert"
	addr := "addr"
	gwAddr := "gwaddr"
	m.nodes[code] = &Node{Code: code}

	// Attempt to insert a node
	err := m.RegisterNode(id.NewIdFromString("", id.Node, t), []byte("test"), code, addr,
		cert, gwAddr, gwCert)

	// Verify the insert was successful
	if info := m.nodes[code]; err != nil || info.NodeCertificate != cert ||
		info.GatewayCertificate != gwCert || info.ServerAddress != addr ||
		info.GatewayAddress != gwAddr {
		t.Errorf("Expected to successfully insert node information: %+v", info)
	}
}

// Error path: Invalid registration code
func TestMapImpl_RegisterNode_Invalid(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	// Do NOT load in a registration code
	code := "TEST"

	// Attempt to insert a node without an associated registration code
	err := m.RegisterNode(id.NewIdFromString("", id.Node, t), []byte("test"), code, code,
		code, code, code)

	// Verify the insert failed
	if err == nil {
		t.Errorf("Expected to fail inserting node information without the" +
			" correct registration code")
	}
}

// Happy path
func TestMapImpl_GetNode(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	// Load in a registration code
	code := "TEST"
	m.nodes[code] = &Node{Code: code}

	// Check that the correct node is obtained
	info, err := m.GetNode(code)
	if err != nil || info.Code != code {
		t.Errorf("Expected to be able to obtain correct node")
	}
}

// Error path: Nonexistent registration code
func TestMapImpl_GetNode_Invalid(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	// Check that no node is obtained from empty map
	info, err := m.GetNode("TEST")
	if err == nil || info != nil {
		t.Errorf("Expected to not find the node")
	}
}

// Happy path
func TestMapImpl_GetNodeById(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	// Load in a registration code
	code := "TEST"
	testId := id.NewIdFromString(code, id.Node, t)
	m.nodes[code] = &Node{Code: code, Id: testId.Marshal()}

	// Check that the correct node is obtained
	info, err := m.GetNodeById(testId)
	if err != nil || info.Code != code {
		t.Errorf("Expected to be able to obtain correct node")
	}
}

// Error path: Nonexistent node id
func TestMapImpl_GetNodeById_Invalid(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	testId := id.NewIdFromString("test", id.Node, t)

	// Check that no node is obtained from empty map
	info, err := m.GetNodeById(testId)
	if err == nil || info != nil {
		t.Errorf("Expected to not find the node")
	}
}

// Happy path
func TestMapImpl_GetNodesByStatus(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	// Should start off empty
	nodes, err := m.GetNodesByStatus(node.Banned)
	if err != nil {
		t.Errorf("Unable to get nodes by status: %+v", err)
	}
	if len(nodes) > 0 {
		t.Errorf("Unexpected nodes returned for status: %v", nodes)
	}

	// Add a banned node
	code := "TEST"
	m.nodes[code] = &Node{Code: code, Status: uint8(node.Banned)}

	// Should have a result now
	nodes, err = m.GetNodesByStatus(node.Banned)
	if err != nil {
		t.Errorf("Unable to get nodes by status: %+v", err)
	}
	if len(nodes) != 1 {
		t.Errorf("Unexpected nodes returned for status: %v", nodes)
	}

	// Unban the node
	m.nodes[code].Status = uint8(node.Active)

	// Shouldn't get a result anymore
	nodes, err = m.GetNodesByStatus(node.Banned)
	if err != nil {
		t.Errorf("Unable to get nodes by status: %+v", err)
	}
	if len(nodes) > 0 {
		t.Errorf("Unexpected nodes returned for status: %v", nodes)
	}
}
