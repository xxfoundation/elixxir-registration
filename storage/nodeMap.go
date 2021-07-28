////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the MapImpl for node-related functionality

package storage

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
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

// Update the address fields for the Node with the given id
func (m *MapImpl) UpdateNodeAddresses(id *id.ID, nodeAddr, gwAddr string) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	for _, v := range m.nodes {
		if bytes.Compare(v.Id, id.Marshal()) == 0 {
			v.GatewayAddress = gwAddr
			v.ServerAddress = nodeAddr
			return nil
		}
	}

	return errors.Errorf("unable to update addresses for %s", id.String())
}

// Update the sequence field for the Node with the given id
func (m *MapImpl) UpdateNodeSequence(id *id.ID, sequence string) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	for _, v := range m.nodes {
		if bytes.Compare(v.Id, id.Marshal()) == 0 {
			v.Sequence = sequence
			return nil
		}
	}

	return errors.Errorf("unable to update sequence for %s", id.String())
}

// If Node registration code is valid, add Node information
func (m *MapImpl) RegisterNode(id *id.ID, salt []byte, code, serverAddress, serverCert,
	gatewayAddress, gatewayCert string) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	jww.INFO.Printf("Attempting to register node with code: %s", code)
	if info := m.nodes[code]; info != nil {
		info.Id = id.Marshal()
		info.Salt = salt
		info.ServerAddress = serverAddress
		info.GatewayCertificate = gatewayCert
		info.GatewayAddress = gatewayAddress
		info.NodeCertificate = serverCert
		info.Status = uint8(node.Active)
		info.DateRegistered = time.Now()
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

// Return all nodes in Storage with the given Status
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

// Return all ActiveNodes in Storage
func (m *MapImpl) GetActiveNodes() ([]*ActiveNode, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	activeNodes := make([]*ActiveNode, 0)
	for _, v := range m.activeNodes {
		activeNodes = append(activeNodes, v)
	}
	return activeNodes, nil
}
