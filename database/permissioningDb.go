////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for the permissioning server

package database

import (
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
)

// If the given node registration code exists,
// insert the provided node information
func (m *DatabaseImpl) InsertNode(code string, id []byte, address, cert string) error {

	// Look up given node registration code
	nodeInfo := NodeInformation{Code: code}
	jww.INFO.Printf("Attempting to register node with code %s...", code)
	err := m.db.Select(&nodeInfo)

	if err != nil {
		// Unable to find code, return error
		return err
	}

	// Update the record with the new node information
	nodeInfo.Id = id
	nodeInfo.Address = address
	nodeInfo.Certificate = cert
	err = m.db.Update(nodeInfo)
	return err
}

// Add the given node registration code to the database
func (m *DatabaseImpl) InsertNodeRegCode(code string) error {
	// Look up given node registration code
	regCode := NodeInformation{Code: code}
	jww.INFO.Printf("Adding node registration code: %s", code)
	err := m.db.Insert(&regCode)
	return err
}

// Obtain the full internal registered node topology
func (m *DatabaseImpl) GetRegisteredNodes() ([]NodeInformation, error) {
	var nodes []NodeInformation
	// Only select Nodes that have already been registered
	err := m.db.Model(&nodes).Where("id IS NOT NULL").Select()
	if err != nil {
		fmt.Printf("%+v", err)
		return nil, err
	}
	return nodes, nil
}
