////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the high level storage API.
// This layer merges the business logic layer and the database layer

package storage

import "testing"

// Global variable for Database interaction
var PermissioningDb Storage

// API for the storage layer
type Storage struct {
	// Stored Database interface
	database
}

// Return GeoBins in Map format from Storage
func (s *Storage) GetBins() (map[string]uint8, error) {
	geoBins, err := s.getBins()
	if err != nil {
		return nil, err
	}

	result := make(map[string]uint8, len(geoBins))
	for _, geoBin := range geoBins {
		result[geoBin.Country] = geoBin.Bin
	}
	return result, nil
}

// Test use only function for exposing MapImpl
func (s *Storage) GetMapImpl(t *testing.T) *MapImpl {
	return s.database.(*MapImpl)
}
