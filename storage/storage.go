////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the high level storage API.
// This layer merges the business logic layer and the database layer

package storage

import (
	"github.com/pkg/errors"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/region"
	"strconv"
	"testing"
	"time"
)

// Global variable for Database interaction
var PermissioningDb Storage

// API for the storage layer
type Storage struct {
	// Stored Database interface
	database
}

// Return GeoBins in Map format from Storage
func (s *Storage) GetBins() (map[string]region.GeoBin, error) {
	geoBins, err := s.getBins()
	if err != nil {
		return nil, err
	}

	result := make(map[string]region.GeoBin, len(geoBins))
	for _, geoBin := range geoBins {
		result[geoBin.Country] = region.GeoBin(geoBin.Bin)
	}
	return result, nil
}

// Set LastActive to now for all the given Nodes in storage
func (s *Storage) UpdateLastActive(ids []*id.ID) error {
	idsBytes := make([][]byte, len(ids))
	for i, nodeId := range ids {
		idsBytes[i] = nodeId.Marshal()
	}
	currentTime := time.Now()

	return s.updateLastActive(idsBytes, currentTime)
}

// Helper for returning a uint64 from the State table
func (s *Storage) GetStateInt(key string) (uint64, error) {
	valueStr, err := s.GetStateValue(key)
	if err != nil {
		return 0, errors.Errorf("Unable to find %s: %+v", key, err)
	}
	value, err := strconv.ParseUint(valueStr, 10, 64)
	if err != nil {
		return 0, errors.Errorf("Unable to decode %s: %+v", valueStr, err)
	}
	return value, nil
}

// Test use only function for exposing MapImpl
func (s *Storage) GetMapImpl(t *testing.T) *MapImpl {
	return s.database.(*MapImpl)
}
