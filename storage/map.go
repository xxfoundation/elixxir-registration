////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Handles Map backend functionality
//go:build stateless
// +build stateless

package storage

// Initialize the Database interface with Database backend
// Returns a Storage interface, Close function, and error
func NewDatabase(username, password, database, address,
	port string) (Storage, func() error, error) {
	return NewMap(), func() error { return nil }, nil
}
