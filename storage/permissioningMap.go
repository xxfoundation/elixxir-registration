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
)

// TODO: Insert Application object along with associated unregistered Node
func (m *MapImpl) InsertApplication(application *Application,
	unregisteredNode *Node) error {
	// TODO: This is map code which will help with inserting the node
	//m.mut.Lock()
	//jww.INFO.Printf("Adding node registration code: %s with Order Info: %s",
	//	code, order)
	//
	//// Enforce unique registration code
	//if m.node[code] != nil {
	//	m.mut.Unlock()
	//	return errors.Errorf("node registration code %s already exists",
	//		code)
	//}
	//
	//m.node[code] = &Node{Code: code, Order: order}
	//m.mut.Unlock()
	return nil
}

// TODO: Insert NodeMetric object
func (m *MapImpl) InsertNodeMetric(metric *NodeMetric) error {
	return nil
}

// TODO: Insert RoundMetric object
func (m *MapImpl) InsertRoundMetric(metric *RoundMetric, topology []string) error {
	return nil
}

// If Node registration code is valid, add Node information
func (m *MapImpl) RegisterNode(id *id.Node, code, serverCert, serverAddress,
	gatewayAddress, gatewayCert string) error {
	m.mut.Lock()
	jww.INFO.Printf("Attempting to register node with code: %s", code)
	if info := m.node[code]; info != nil {
		info.Id = id.String()
		info.ServerAddress = serverAddress
		info.GatewayCertificate = gatewayCert
		info.GatewayAddress = gatewayAddress
		info.NodeCertificate = serverCert
		m.mut.Unlock()
		return nil
	}
	m.mut.Unlock()
	return errors.Errorf("unable to register node %s", code)

}

// Get Node information for the given Node registration code
func (m *MapImpl) GetNode(code string) (*Node, error) {
	m.mut.Lock()
	info := m.node[code]
	if info == nil {
		m.mut.Unlock()
		return nil, errors.Errorf("unable to get node %s", code)
	}
	m.mut.Unlock()
	return info, nil
}
