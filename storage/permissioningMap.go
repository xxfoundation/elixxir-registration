////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the Map backend for the permissioning server

package storage

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Insert Application object along with associated unregistered Node
func (m *MapImpl) InsertApplication(application *Application, unregisteredNode *Node) error {
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

	m.nodes[unregisteredNode.Code] = unregisteredNode
	m.applications[application.Id] = application
	return nil
}

// Insert NodeMetric object
func (m *MapImpl) InsertNodeMetric(metric *NodeMetric) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	// Auto-increment key
	m.nodeMetricCounter += 1

	// Add to map
	metric.Id = m.nodeMetricCounter
	jww.DEBUG.Printf("Attempting to insert node metric: %+v", metric)
	m.nodeMetrics[m.nodeMetricCounter] = metric
	return nil
}

// Insert RoundError object
func (m *MapImpl) InsertRoundError(roundId id.Round, errStr string) error {
	m.mut.Lock()
	defer m.mut.Unlock()
	rid := uint64(roundId)

	m.roundMetrics[rid].RoundErrors = append(
		m.roundMetrics[rid].RoundErrors,
		RoundError{
			Id:            0, // Currently useless in MapImpl
			RoundMetricId: rid,
			Error:         errStr,
		},
	)
	return nil
}

// Insert RoundMetric object with associated topology
func (m *MapImpl) InsertRoundMetric(metric *RoundMetric, topology [][]byte) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	// Build Topology objects
	metric.Topologies = make([]Topology, len(topology))
	for i, nodeIdBytes := range topology {
		nodeId, err := id.Unmarshal(nodeIdBytes)
		if err != nil {
			return errors.New(err.Error())
		}
		topologyObj := Topology{
			NodeId:        nodeId.Bytes(),
			RoundMetricId: metric.Id,
			Order:         uint8(i),
		}
		metric.Topologies[i] = topologyObj
	}

	// Add to map
	jww.DEBUG.Printf("Attempting to insert round metric: %+v", metric)
	m.roundMetrics[metric.Id] = metric
	return nil
}

// If Node registration code is valid, add Node information
func (m *MapImpl) RegisterNode(id *id.ID, code, serverAddress, serverCert,
	gatewayAddress, gatewayCert string) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	jww.INFO.Printf("Attempting to register node with code: %s", code)
	if info := m.nodes[code]; info != nil {
		info.Id = id.Marshal()
		info.ServerAddress = serverAddress
		info.GatewayCertificate = gatewayCert
		info.GatewayAddress = gatewayAddress
		info.NodeCertificate = serverCert
		info.Status = uint8(node.Active)
		return nil
	}
	return errors.Errorf("unable to register node %s", code)

}

// Get Node information for the given Node registration code
func (m *MapImpl) GetNode(code string) (*Node, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	info := m.nodes[code]
	if info == nil {
		return nil, errors.Errorf("unable to get node %s", code)
	}
	return info, nil
}

// Get Node information for the given Node ID
func (m *MapImpl) GetNodeById(id *id.ID) (*Node, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	for _, v := range m.nodes {
		if bytes.Compare(v.Id, id.Marshal()) == 0 {
			return v, nil
		}
	}
	return nil, errors.Errorf("unable to get node %s", id.String())
}

// Return all nodes in storage with the given Status
func (m *MapImpl) GetNodesByStatus(status node.Status) ([]*Node, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	nodes := make([]*Node, 0)
	for _, v := range m.nodes {
		if node.Status(v.Status) == status {
			nodes = append(nodes, v)
		}
	}
	return nodes, nil
}

// If Node registration code is valid, add Node information
func (m *MapImpl) BannedNode(id *id.ID, t interface{}) error {
	// Ensure we're called from a test only
	switch t.(type) {
	case *testing.T:
	case *testing.M:
	case *testing.B:
	default:
		jww.FATAL.Panicf("BannedNode permissioning map function called outside testing")
	}

	m.mut.Lock()
	defer m.mut.Unlock()
	for _, n := range m.nodes {
		if bytes.Compare(n.Id, id.Bytes()) == 0 {
			n.Status = uint8(node.Banned)
			return nil
		}
	}
	return errors.New("Node could not be found in map")
}
