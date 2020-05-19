////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for nodes

package storage

import (
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage/node"
	"time"
)

// Insert Application object along with associated unregistered Node
func (m *DatabaseImpl) InsertApplication(application Application, unregisteredNode Node) error {
	application.Node = unregisteredNode
	return m.db.Create(application).Error
}

// Insert NodeMetric object
func (m *DatabaseImpl) InsertNodeMetric(metric NodeMetric) error {
	return m.db.Create(metric).Error
}

// Insert RoundMetric object
func (m *DatabaseImpl) InsertRoundMetric(metric RoundMetric, topology []string) error {
	newTopology := make([]Topology, len(topology))
	for i, node := range topology {
		topologyObj := Topology{
			NodeId: node,
			Order:  uint8(i),
		}
		newTopology[i] = topologyObj
	}
	metric.Topologies = newTopology
	return m.db.Save(metric).Error
}

// If Node registration code is valid, add Node information
func (m *DatabaseImpl) RegisterNode(id *id.ID, code, serverAddr, serverCert,
	gatewayAddress, gatewayCert string) error {
	newNode := Node{
		Code:               code,
		Id:                 id.Marshal(),
		ServerAddress:      serverAddr,
		GatewayAddress:     gatewayAddress,
		NodeCertificate:    serverCert,
		GatewayCertificate: gatewayCert,
		Status:             uint8(node.Active),
		DateRegistered:     time.Now(),
	}
	return m.db.Model(&newNode).Update(&newNode).Error
}

// Get Node information for the given Node registration code
func (m *DatabaseImpl) GetNode(code string) (*Node, error) {
	newNode := &Node{}
	err := m.db.First(&newNode, "code = ?", code).Error
	return newNode, err
}

// Return all nodes in storage with the given Status
func (m *DatabaseImpl) GetNodesByStatus(status node.Status) ([]*Node, error) {
	var nodes []*Node
	err := m.db.Where("status = ?", status).Find(&nodes).Error
	return nodes, err
}
