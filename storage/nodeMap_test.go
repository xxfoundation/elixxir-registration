////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/region"
	"strconv"
	"strings"
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
		Sequence:      region.Americas.String(),
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
		Sequence:      region.MiddleEast.String(),
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

// Happy path
func TestMapImpl_UpdateNodeAddresses(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	testString := "test"
	testId := id.NewIdFromString(testString, id.Node, t)
	testResult := "newAddr"
	m.nodes[testString] = &Node{
		Code:           testString,
		Id:             testId.Marshal(),
		ServerAddress:  testString,
		GatewayAddress: testString,
	}

	err := m.UpdateNodeAddresses(testId, testResult, testResult)
	if err != nil {
		t.Errorf(err.Error())
	}

	if result := m.nodes[testString]; result.ServerAddress != testResult || result.GatewayAddress != testResult {
		t.Errorf("Field values did not update correctly, got Node %s Gateway %s",
			result.ServerAddress, result.GatewayAddress)
	}
}

// Happy path
func TestMapImpl_UpdateSequence(t *testing.T) {
	m := &MapImpl{
		nodes: make(map[string]*Node),
	}

	testString := region.Americas.String()
	testId := id.NewIdFromString(testString, id.Node, t)
	testResult := "newAddr"
	m.nodes[testString] = &Node{
		Code:           testString,
		Id:             testId.Marshal(),
		Sequence:       testString,
		ServerAddress:  testString,
		GatewayAddress: testString,
	}

	err := m.UpdateNodeSequence(testId, testResult)
	if err != nil {
		t.Errorf(err.Error())
	}

	if result := m.nodes[testString]; result.Sequence != testResult {
		t.Errorf("Sequence values did not update correctly, got %s expected %s",
			result.Sequence, testResult)
	}
}

// Unit test
func TestMapImpl_GetBin(t *testing.T) {
	// Set up and populate the map with testing values
	m := &MapImpl{
		geographicBin: make(map[string]uint8),
	}
	testStrings := []string{"0", "1", "2", "3", "4"}
	expectedBins := make([]uint8, 0, len(testStrings))
	for _, s := range testStrings {
		bin, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("Failed on setup: %v", err)
		}
		m.geographicBin[s] = uint8(bin)

		expectedBins = append(expectedBins, uint8(bin))
	}

	// Test that it pulls values as expected
	for i, s := range testStrings {
		received, err := m.GetBin(s)
		if err != nil {
			t.Errorf("Failed to retrieved bin from map: %v", err)
		}

		if strings.Compare(strconv.Itoa(int(received)), strconv.Itoa(int(expectedBins[i]))) != 0 {
			t.Errorf("Unexpected bin with country code %s. "+
				"\n\tExpected: %v"+
				"\n\tReceived: %v", s, strconv.Itoa(int(expectedBins[i])), strconv.Itoa(int(received)))
		}

	}

	// Failure case: attempt to get a bin from an invalid country code
	_, err := m.GetBin("GraetBritain")
	if err == nil {
		t.Fatalf("Expected failure case. Should not return bin for unpopulated country code!")
	}

}
