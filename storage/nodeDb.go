////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the DatabaseImpl for node-related functionality
//+build !stateless

package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Insert Application object along with associated unregistered Node
func (d *DatabaseImpl) InsertApplication(application *Application, unregisteredNode *Node) error {
	application.Node = *unregisteredNode
	return d.db.Create(application).Error
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

// Update the sequence field for the Node with the given id
func (d *DatabaseImpl) UpdateNodeSequence(id *id.ID, sequence string) error {
	newNode := Node{
		Sequence: sequence,
	}
	return d.db.Take(&newNode, "id = ?", id.Marshal()).Update("sequence", sequence).Error
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
	err := d.db.Take(&newNode, "code = ?", code).Error
	return newNode, err
}

// Get Node information for the given Node ID
func (d *DatabaseImpl) GetNodeById(id *id.ID) (*Node, error) {
	newNode := &Node{}
	err := d.db.Take(&newNode, "id = ?", id.Marshal()).Error
	return newNode, err
}

// Return all nodes in Storage with the given Status
func (d *DatabaseImpl) GetNodesByStatus(status node.Status) ([]*Node, error) {
	var nodes []*Node
	err := d.db.Where("status = ?", uint8(status)).Find(&nodes).Error
	jww.INFO.Printf("GetNodesByStatus: Got %d nodes with status "+
		"%s(%d) from the database", len(nodes), status, status)
	return nodes, err
}

// Return all ActiveNodes in Storage
func (d *DatabaseImpl) GetActiveNodes() ([]*ActiveNode, error) {
	var activeNodes []*ActiveNode
	err := d.db.Find(&activeNodes).Error
	return activeNodes, err
}

// Return the corresponding Bin for the given countryCode
func (d *DatabaseImpl) GetBin(countryCode string) (uint8, error) {
	result := &GeoBin{}
	err := d.db.Take(&result, "country = ?", countryCode).Error
	return result.Bin, err
}
