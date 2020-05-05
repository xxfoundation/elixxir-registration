////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"gitlab.com/elixxir/primitives/id"
	"testing"
)

// Happy path
func TestMapImpl_InsertNodeRegCode(t *testing.T) {
	m := &MapImpl{
		nodes:        make(map[string]*Node),
		applications: make(map[uint64]*Application),
	}

	// Attempt to load in a valid code
	applicationId := uint64(10)
	newNode := Node{
		Code:          "TEST",
		Order:         "BLARG",
		ApplicationId: applicationId,
	}
	newApplication := Application{Id: applicationId}
	err := m.InsertApplication(newApplication, newNode)

	// Verify the insert was successful
	if err != nil || m.nodes[newNode.Code] == nil {
		t.Errorf("Expected to successfully insert node registration code")
	}

	if m.nodes[newNode.Code].Order != newNode.Order {
		t.Errorf("Order string incorret; Expected: %s, Recieved: %s",
			newNode.Order, m.nodes[newNode.Code].Order)
	}
}

// Error Path: Duplicate node registration code
func TestMapImpl_InsertNodeRegCode_Duplicate(t *testing.T) {
	m := &MapImpl{
		nodes:        make(map[string]*Node),
		applications: make(map[uint64]*Application),
	}

	// Load in a registration code
	applicationId := uint64(10)
	newNode := Node{
		Code:          "TEST",
		Order:         "BLARG",
		ApplicationId: applicationId,
	}
	newApplication := Application{Id: applicationId}
	m.nodes[newNode.Code] = &newNode

	// Attempt to load in a duplicate code
	err := m.InsertApplication(newApplication, newNode)

	// Verify the insert failed
	if err == nil {
		t.Errorf("Expected to fail inserting duplicate node registration code")
	}
}

// Happy path
func TestMapImpl_InsertNode(t *testing.T) {
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
	err := m.RegisterNode(id.NewNodeFromBytes(make([]byte, 0)), code, cert,
		addr, gwAddr, gwCert)

	// Verify the insert was successful
	if info := m.nodes[code]; err != nil || info.NodeCertificate != cert ||
		info.GatewayCertificate != gwCert || info.ServerAddress != addr ||
		info.GatewayAddress != gwAddr {
		t.Errorf("Expected to successfully insert node information: %+v", info)
	}
}

// Error path: Invalid registration code
func TestMapImpl_InsertNode_Invalid(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	// Do NOT load in a registration code
	code := "TEST"

	// Attempt to insert a node without an associated registration code
	err := m.RegisterNode(id.NewNodeFromBytes(make([]byte, 0)), code, code,
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
func TestMapImpl_InsertUser(t *testing.T) {
	m := &MapImpl{
		users: make(map[string]bool),
	}

	testKey := "TEST"
	_ = m.InsertUser(testKey)
	if !m.users[testKey] {
		t.Errorf("Insert failed to add the user!")
	}
}

// Happy path
func TestMapImpl_GetUser(t *testing.T) {
	m := &MapImpl{
		users: make(map[string]bool),
	}

	testKey := "TEST"
	m.users[testKey] = true

	user, err := m.GetUser(testKey)
	if err != nil || user.PublicKey != testKey {
		t.Errorf("Get failed to get user!")
	}
}

// Get user that does not exist
func TestMapImpl_GetUserNotExists(t *testing.T) {
	m := &MapImpl{
		users: make(map[string]bool),
	}

	testKey := "TEST"

	_, err := m.GetUser(testKey)
	if err == nil {
		t.Errorf("Get expected to not find user!")
	}
}
