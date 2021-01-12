////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles Map backend functionality
//+build !database,!darwin,!linux,!windows

package storage

import (
	jww "github.com/spf13/jwalterweatherman"
)

// Initialize the Database interface with Database backend
// Returns a Storage interface, Close function, and error
func NewDatabase(username, password, database, address,
	port string) (Storage, func() error, error) {
	defer jww.INFO.Println("Map backend initialized successfully!")
	return Storage{
		&MapImpl{
			applications: make(map[uint64]*Application),
			nodes:        make(map[string]*Node),
			nodeMetrics:  make(map[uint64]*NodeMetric),
			roundMetrics: make(map[uint64]*RoundMetric),
			clients:      make(map[string]*RegistrationCode),
			users:        make(map[string]bool),
			states:       make(map[string]string),
		}}, func() error { return nil }, nil
}
