////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage/node"
	"testing"
	"time"
)

// Happy path
func TestMapImpl_InsertNodeMetric(t *testing.T) {
	m := &MapImpl{nodeMetrics: make(map[uint64]*NodeMetric)}

	newMetric := NodeMetric{
		NodeId:    "TEST",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		NumPings:  1000,
	}

	err := m.InsertNodeMetric(newMetric)
	if err != nil {
		t.Errorf("Unable to insert node metric: %+v", err)
	}

	insertedMetric := m.nodeMetrics[m.nodeMetricCounter]
	if insertedMetric.Id != m.nodeMetricCounter {
		t.Errorf("Mismatched ID returned!")
	}
	if insertedMetric.StartTime != newMetric.StartTime {
		t.Errorf("Mismatched StartTime returned!")
	}
	if insertedMetric.EndTime != newMetric.EndTime {
		t.Errorf("Mismatched EndTime returned!")
	}
	if insertedMetric.NumPings != newMetric.NumPings {
		t.Errorf("Mismatched NumPings returned!")
	}
}

// Happy path
func TestMapImpl_InsertRoundMetric(t *testing.T) {
	m := &MapImpl{roundMetrics: make(map[uint64]*RoundMetric)}

	newMetric := RoundMetric{
		Error:         "TEST",
		PrecompStart:  time.Now(),
		PrecompEnd:    time.Now(),
		RealtimeStart: time.Now(),
		RealtimeEnd:   time.Now(),
		BatchSize:     420,
	}
	newTopology := [][]byte{id.NewIdFromBytes([]byte("Node1"), t).Bytes(),
		id.NewIdFromBytes([]byte("Node2"), t).Bytes()}

	err := m.InsertRoundMetric(newMetric, newTopology)
	if err != nil {
		t.Errorf("Unable to insert round metric: %+v", err)
	}

	insertedMetric := m.roundMetrics[m.roundMetricCounter]
	if insertedMetric.Id != m.roundMetricCounter {
		t.Errorf("Mismatched ID returned!")
	}
	if insertedMetric.PrecompStart != newMetric.PrecompStart {
		t.Errorf("Mismatched PrecompStart returned!")
	}
	if insertedMetric.PrecompEnd != newMetric.PrecompEnd {
		t.Errorf("Mismatched PrecompEnd returned!")
	}
	if insertedMetric.RealtimeStart != newMetric.RealtimeStart {
		t.Errorf("Mismatched RealtimeStart returned!")
	}
	if insertedMetric.RealtimeEnd != newMetric.RealtimeEnd {
		t.Errorf("Mismatched RealtimeEnd returned!")
	}
	if insertedMetric.BatchSize != newMetric.BatchSize {
		t.Errorf("Mismatched BatchSize returned!")
	}
	if insertedMetric.Error != newMetric.Error {
		t.Errorf("Mismatched Error returned!")
	}
}

// Happy path
func TestMapImpl_InsertApplication(t *testing.T) {
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

// Error Path: Duplicate node registration code and application
func TestMapImpl_InsertApplication_Duplicate(t *testing.T) {
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

	// Attempt to load in a duplicate application
	m.applications[applicationId] = &newApplication
	err := m.InsertApplication(newApplication, newNode)

	// Verify the insert failed
	if err == nil {
		t.Errorf("Expected to fail inserting duplicate application")
	}

	// Attempt to load in a duplicate code
	m.nodes[newNode.Code] = &newNode
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
	err := m.RegisterNode(id.NewIdFromString("", id.Node, t), code, addr,
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
	err := m.RegisterNode(id.NewIdFromString("", id.Node, t), code, code,
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
