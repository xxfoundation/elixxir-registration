////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for nodes

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage/node"
	"time"
)

// Insert Application object along with associated unregistered Node
func (m *DatabaseImpl) InsertApplication(application *Application, unregisteredNode *Node) error {
	application.Node = *unregisteredNode
	return m.db.Create(application).Error
}

// Insert NodeMetric object
func (m *DatabaseImpl) InsertNodeMetric(metric *NodeMetric) error {
	jww.DEBUG.Printf("Attempting to insert node metric: %+v", metric)
	return m.db.Create(metric).Error
}

// Insert RoundError object
func (m *DatabaseImpl) InsertRoundError(roundId uint64, errStr string) error {
	roundErr := &RoundError{
		RoundMetricId: roundId,
		Error:         errStr,
	}
	jww.DEBUG.Printf("Attempting to insert round error: %+v", roundErr)
	return m.db.Create(roundErr).Error
}

// Insert RoundMetric object with associated topology and errors
func (m *DatabaseImpl) InsertRoundMetric(metric *RoundMetric, topology [][]byte) error {

	// Build the Topology
	metric.Topologies = make([]Topology, len(topology))
	for i, nodeIdBytes := range topology {
		nodeId, err := id.Unmarshal(nodeIdBytes)
		if err != nil {
			return errors.New(err.Error())
		}
		topologyObj := Topology{
			NodeId: nodeId.Bytes(),
			Order:  uint8(i),
		}
		metric.Topologies[i] = topologyObj
	}

	// Save the RoundMetric
	jww.DEBUG.Printf("Attempting to insert round metric: %+v", metric)
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
	err := m.db.Where("status = ?", uint8(status)).Find(&nodes).Error
	return nodes, err
}
