////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for the permissioning server

package database

import (
	jww "github.com/spf13/jwalterweatherman"
)

// If the given Node registration code exists,
// insert the provided Node information
func (m *DatabaseImpl) InsertNode(id []byte, code, serverCert,
	gatewayAddress, gatewayCert string) error {
	// Look up given node registration code
	nodeInfo := NodeInformation{Code: code}
	jww.INFO.Printf("Attempting to register node with code: %s", code)
	err := m.db.Select(&nodeInfo)

	if err != nil {
		// Unable to find code, return error
		return err
	}

	// Update the record with the new node information
	nodeInfo.Id = id
	nodeInfo.NodeCertificate = serverCert
	nodeInfo.GatewayAddress = gatewayAddress
	nodeInfo.GatewayCertificate = gatewayCert
	err = m.db.Update(&nodeInfo)
	return err
}

// Add the given Node registration code to the database
func (m *DatabaseImpl) InsertNodeRegCode(code string) error {
	regCode := NodeInformation{Code: code}
	jww.INFO.Printf("Adding node registration code: %s", code)
	err := m.db.Insert(&regCode)
	return err
}

// Count the number of Nodes currently registered
func (m *DatabaseImpl) CountRegisteredNodes() (int, error) {
	var nodes []NodeInformation
	// Only count Nodes that have already been registered
	return m.db.Model(&nodes).Where("id IS NOT NULL").Count()
}

// Get Node information for the given Node registration code
func (m *DatabaseImpl) GetNode(code string) (*NodeInformation, error) {
	node := &NodeInformation{Code: code}
	err := m.db.Select(node)
	return node, err
}
