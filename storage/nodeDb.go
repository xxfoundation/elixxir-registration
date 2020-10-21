////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the DatabaseImpl for node-related functionality

package storage

import (
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Insert Application object along with associated unregistered Node
func (m *DatabaseImpl) InsertApplication(application *Application, unregisteredNode *Node) error {
	application.Node = *unregisteredNode
	return m.db.Create(application).Error
}

// Update the Salt for a given Node ID
func (m *DatabaseImpl) UpdateSalt(id *id.ID, salt []byte) error {
	newNode := Node{
		Salt: salt,
	}
	return m.db.First(&newNode, "id = ?", id.Marshal()).Update("salt", salt).Error
}

// If Node registration code is valid, add Node information
func (m *DatabaseImpl) RegisterNode(id *id.ID, salt []byte, code, serverAddr, serverCert,
	gatewayAddress, gatewayCert string) error {
	newNode := Node{
		Code:               code,
		Id:                 id.Marshal(),
		Salt:               salt,
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

// Get Node information for the given Node ID
func (m *DatabaseImpl) GetNodeById(id *id.ID) (*Node, error) {
	newNode := &Node{}
	err := m.db.First(&newNode, "id = ?", id.Marshal()).Error
	return newNode, err
}

// Return all nodes in storage with the given Status
func (m *DatabaseImpl) GetNodesByStatus(status node.Status) ([]*Node, error) {
	var nodes []*Node
	err := m.db.Where("status = ?", uint8(status)).Find(&nodes).Error
	return nodes, err
}
