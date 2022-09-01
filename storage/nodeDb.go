////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Handles the DatabaseImpl for node-related functionality
//go:build !stateless
// +build !stateless

package storage

import (
	"github.com/pkg/errors"
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

// Update the given applicationId with the given GeoIP information
func (d *DatabaseImpl) UpdateGeoIP(appId uint64, location, geoBin, gpsLocation string) error {
	app := &Application{
		Id: appId,
	}
	err := d.db.First(&app).Error
	if err != nil {
		return errors.WithMessagef(err, "Failed to find application with id %d", appId)
	}

	app.GeoBin = geoBin
	app.GpsLocation = gpsLocation
	app.Location = location

	err = d.db.Save(&app).Error
	if err != nil {
		return errors.WithMessagef(err, "Failed to update geo info for app id %d", appId)
	}
	return nil
}

// Update LastActive field for all given Node IDs in Storage
func (d *DatabaseImpl) updateLastActive(ids [][]byte, lastActive time.Time) error {
	return d.db.Model(Node{}).Where("id IN (?)", ids).
		Update("last_active", lastActive).Error
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

// Return all nodes in Storage
func (d *DatabaseImpl) GetNodes() ([]*Node, error) {
	var nodes []*Node
	err := d.db.Find(&nodes).Error
	return nodes, err
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
