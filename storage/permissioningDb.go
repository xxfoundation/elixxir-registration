////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for nodes

package storage

import (
	"gitlab.com/elixxir/primitives/id"
	"time"
)

// If Node registration code is valid, add Node information
func (m *DatabaseImpl) InsertNode(id *id.Node, code, serverAddr, serverCert,
	gatewayAddress, gatewayCert string) error {
	newNode := Node{
		Code:               code,
		Id:                 id.String(),
		ServerAddress:      serverAddr,
		GatewayAddress:     gatewayAddress,
		NodeCertificate:    serverCert,
		GatewayCertificate: gatewayCert,
		DateRegistered:     time.Now(),
	}
	return m.db.Create(&newNode).Error
}

// Insert Node registration code into the database
func (m *DatabaseImpl) InsertNodeRegCode(regCode, order string) error {
	newNode := &Node{
		Code:  regCode,
		Order: order,
	}
	return m.db.Create(newNode).Error
}

// Get Node information for the given Node registration code
func (m *DatabaseImpl) GetNode(code string) (*Node, error) {
	newNode := &Node{}
	err := m.db.First(&newNode, "code = ?", code).Error
	return newNode, err
}
