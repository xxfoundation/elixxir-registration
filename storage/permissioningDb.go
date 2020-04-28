////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for nodes

package storage

import (
	"time"
)

// If Node registration code is valid, add Node information
func (m *DatabaseImpl) InsertNode(id []byte, code, serverAddr, serverCert,
	gatewayAddress, gatewayCert string) error {
	newNode := Node{
		Code:               code,
		Id:                 string(id),
		ServerAddress:      serverAddr,
		GatewayAddress:     gatewayAddress,
		NodeCertificate:    serverCert,
		GatewayCertificate: gatewayCert,
		DateRegistered:     time.Now(),
	}
	return m.db.Update(&newNode)
}

// Insert Node registration code into the database
func (m *DatabaseImpl) InsertNodeRegCode(regCode, order string) error {
	newNode := Node{
		Code:  regCode,
		Order: order,
	}
	return m.db.Insert(&newNode)
}

// Get Node information for the given Node registration code
func (m *DatabaseImpl) GetNode(code string) (*Node, error) {
	newNode := &Node{
		Code: code,
	}
	err := m.db.Select(newNode)
	return newNode, err
}
