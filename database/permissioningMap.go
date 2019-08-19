////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the Map backend for the permissioning server

package database

import (
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
)

// If Node registration code is valid, add Node information
func (m *MapImpl) InsertNode(id []byte, code, address,
	nodeCert, gatewayCert string) error {
	jww.INFO.Printf("Attempting to register node with code: %s", code)
	if info := m.node[code]; info != nil {
		info.Id = id
		info.GatewayCertificate = gatewayCert
		info.NodeCertificate = nodeCert
		info.Address = address
		return nil
	}

	return errors.New(fmt.Sprintf("unable to register node %s", code))

}

// Insert Node registration code into the database
func (m *MapImpl) InsertNodeRegCode(code string) error {
	jww.INFO.Printf("Adding node registration code: %s", code)

	// Enforce unique registration code
	if m.node[code] != nil {
		return errors.New(fmt.Sprintf(
			"node registration code %s already exists", code))
	}

	m.node[code] = &NodeInformation{Code: code}
	return nil
}

// Count the number of Nodes currently registered
func (m *MapImpl) CountRegisteredNodes() (int, error) {
	counter := 0
	for _, v := range m.node {
		if v.Id != nil {
			counter += 1
		}
	}
	return counter, nil
}

// Get Node information for the given Node registration code
func (m *MapImpl) GetNode(code string) (*NodeInformation, error) {
	info := m.node[code]
	if info == nil {
		return nil, errors.New(fmt.Sprintf("unable to get node %s", code))
	}
	return info, nil
}

//Delete a code from being able to be used again
func (m *MapImpl) DeleteCode(code string) error {
	m.client[code] = nil
	return nil
}
