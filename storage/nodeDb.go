////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the DatabaseImpl for node-related functionality
//+build !stateless

package storage

import (
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"time"
	jww "github.com/spf13/jwalterweatherman"
)

// Insert Application object along with associated unregistered Node
func (d *DatabaseImpl) InsertApplication(application *Application, unregisteredNode *Node) error {
	application.Node = *unregisteredNode
	return d.db.Create(application).Error
}

// Update the Salt for a given Node ID
func (d *DatabaseImpl) UpdateSalt(id *id.ID, salt []byte) error {
	newNode := Node{
		Salt: salt,
	}
	return d.db.First(&newNode, "id = ?", id.Marshal()).Update("salt", salt).Error
}

// If Node registration code is valid, add Node information
func (d *DatabaseImpl) RegisterNode(id *id.ID, salt []byte, code, serverAddr, serverCert,
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
	return d.db.Model(&newNode).Update(&newNode).Error
}

// Get Node information for the given Node registration code
func (d *DatabaseImpl) GetNode(code string) (*Node, error) {
	newNode := &Node{}
	err := d.db.First(&newNode, "code = ?", code).Error
	return newNode, err
}

// Get Node information for the given Node ID
func (d *DatabaseImpl) GetNodeById(id *id.ID) (*Node, error) {
	newNode := &Node{}
	err := d.db.First(&newNode, "id = ?", id.Marshal()).Error
	return newNode, err
}

// Return all nodes in storage with the given Status
func (d *DatabaseImpl) GetNodesByStatus(status node.Status) ([]*Node, error) {
	var nodes []*Node
	err := d.db.Where("status = ?", uint8(status)).Find(&nodes).Error
	jww.INFO.Printf("GetNodesByStatus: Got %d nodes with status " +
		"%s(%d) from the database", len(nodes), status, status)
	return nodes, err
}

// Update the address fields for the Node with the given id
func (d *DatabaseImpl) UpdateNodeAddresses(id *id.ID, nodeAddr, gwAddr string) error {
	newNode := &Node{
		Id:             id.Marshal(),
		ServerAddress:  nodeAddr,
		GatewayAddress: gwAddr,
	}
	return d.db.Model(newNode).Where("id = ?", newNode.Id).Updates(map[string]interface{}{
		"server_address":  nodeAddr,
		"gateway_address": gwAddr,
	}).Error
}
