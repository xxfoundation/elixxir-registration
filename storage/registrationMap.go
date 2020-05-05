////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the Map backend for registration codes

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

// Inserts Client registration code with given number of uses
func (m *MapImpl) InsertClientRegCode(code string, uses int) error {
	m.mut.Lock()
	jww.INFO.Printf("Inserting code %s, %d uses remaining", code, uses)
	// Enforce unique registration code
	if m.clients[code] != nil {
		m.mut.Unlock()
		return errors.Errorf("client registration code %s already exists", code)
	}
	m.clients[code] = &RegistrationCode{
		Code:          code,
		RemainingUses: uses,
	}
	m.mut.Unlock()
	return nil
}

// If Client registration code is valid, decrements remaining uses
func (m *MapImpl) UseCode(code string) error {
	m.mut.Lock()
	// Look up given registration code
	jww.INFO.Printf("Attempting to use code %s...", code)
	reg := m.clients[code]
	if reg == nil {
		// Unable to find code, return error
		m.mut.Unlock()
		return errors.Errorf("invalid registration code")
	}

	if reg.RemainingUses < 1 {
		// Code has no remaining uses, return error
		m.mut.Unlock()
		return errors.Errorf("registration code %s has no remaining uses", code)
	}

	// Decrement remaining uses by one
	reg.RemainingUses -= 1
	jww.INFO.Printf("Code %s used, %d uses remaining", code,
		reg.RemainingUses)
	m.mut.Unlock()
	return nil
}

// Gets User from the map
func (m *MapImpl) GetUser(publicKey string) (*User, error) {
	if ok := m.users[publicKey]; ok {
		return &User{
			PublicKey: publicKey,
		}, nil
	}
	return nil, errors.New("user does not exist")
}

// Inserts User into the map
func (m *MapImpl) InsertUser(publicKey string) error {
	m.users[publicKey] = true
	return nil
}
