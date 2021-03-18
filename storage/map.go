////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles Map backend functionality
//+build stateless

package storage

// Initialize the Database interface with Database backend
// Returns a Storage interface, Close function, and error
func NewDatabase(username, password, database, address,
	port string) (Storage, func() error, error) {
	return NewMap(), func() error { return nil }, nil
}
