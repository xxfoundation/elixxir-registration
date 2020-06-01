////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the Map backend for the permissioning server

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage/node"
)

// Insert Application object along with associated unregistered Node
func (m *MapImpl) InsertApplication(application Application,
	unregisteredNode Node) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	jww.INFO.Printf("Adding application: %d", application.Id)
	jww.INFO.Printf("Adding node registration code: %s with Order Info: %s",
		unregisteredNode.Code, unregisteredNode.Sequence)

	// Enforce unique keys
	if m.nodes[unregisteredNode.Code] != nil {
		return errors.Errorf("node registration code %s already exists",
			unregisteredNode.Code)
	}
	if m.applications[application.Id] != nil {
		return errors.Errorf("application ID %d already exists",
			application.Id)
	}

	m.nodes[unregisteredNode.Code] = &unregisteredNode
	m.applications[application.Id] = &application
	return nil
}

// Insert NodeMetric object
func (m *MapImpl) InsertNodeMetric(metric NodeMetric) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	// Auto-increment key
	m.nodeMetricCounter += 1

	// Add to map
	metric.Id = m.nodeMetricCounter
	jww.DEBUG.Printf("Attempting to insert node metric: %+v", metric)
	m.nodeMetrics[m.nodeMetricCounter] = &metric
	return nil
}

// Insert RoundMetric object
func (m *MapImpl) InsertRoundMetric(metric RoundMetric, topology [][]byte) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	// Auto-increment key
	m.roundMetricCounter += 1
	metric.Id = m.roundMetricCounter

	// Build Topology objects
	metric.Topologies = make([]Topology, len(topology))
	for i, node := range topology {
		nodeId, err := id.Unmarshal(node)
		if err != nil {
			return errors.New(err.Error())
		}
		topologyObj := Topology{
			NodeId:        nodeId.Bytes(),
			RoundMetricId: m.roundMetricCounter,
			Order:         uint8(i),
		}
		metric.Topologies[i] = topologyObj
	}

	// Add to map
	jww.DEBUG.Printf("Attempting to insert round metric: %+v", metric)
	m.roundMetrics[m.roundMetricCounter] = &metric
	return nil
}

// If Node registration code is valid, add Node information
func (m *MapImpl) RegisterNode(id *id.ID, code, serverCert, serverAddress,
	gatewayAddress, gatewayCert string) error {
	m.mut.Lock()
	jww.INFO.Printf("Attempting to register node with code: %s", code)
	if info := m.nodes[code]; info != nil {
		info.Id = id.Marshal()
		info.ServerAddress = serverAddress
		info.GatewayCertificate = gatewayCert
		info.GatewayAddress = gatewayAddress
		info.NodeCertificate = serverCert
		info.Status = uint8(node.Active)
		m.mut.Unlock()
		return nil
	}
	m.mut.Unlock()
	return errors.Errorf("unable to register node %s", code)

}

// Get Node information for the given Node registration code
func (m *MapImpl) GetNode(code string) (*Node, error) {
	m.mut.Lock()
	info := m.nodes[code]
	if info == nil {
		m.mut.Unlock()
		return nil, errors.Errorf("unable to get node %s", code)
	}
	m.mut.Unlock()
	return info, nil
}

// Return all nodes in storage with the given Status
func (m *MapImpl) GetNodesByStatus(status node.Status) ([]*Node, error) {
	nodes := make([]*Node, 0)
	for _, v := range m.nodes {
		if node.Status(v.Status) == status {
			nodes = append(nodes, v)
		}
	}
	return nodes, nil
}
