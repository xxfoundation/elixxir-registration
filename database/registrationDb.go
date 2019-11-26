////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for registration codes

package database

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
)

// Inserts client registration code with given number of uses
func (m *DatabaseImpl) InsertClientRegCode(code string, uses int) error {
	jww.INFO.Printf("Inserting code %s, %d uses remaining", code, uses)
	err := m.db.Insert(&RegistrationCode{
		Code:          code,
		RemainingUses: uses,
	})
	return err
}

// If client registration code is valid, decrements remaining uses
func (m *DatabaseImpl) UseCode(code string) error {

	// Look up given registration code
	regCode := RegistrationCode{Code: code}
	jww.INFO.Printf("Attempting to use code %s...", code)
	err := m.db.Select(&regCode)
	if err != nil {
		// Unable to find code, return error
		return err
	}

	if regCode.RemainingUses < 1 {
		// Code has no remaining uses, return error
		return errors.Errorf("Code %s has no remaining uses", code)
	}

	// Decrement remaining uses by one
	regCode.RemainingUses -= 1
	err = m.db.Update(&regCode)

	jww.INFO.Printf("Code %s used, %d uses remaining", code,
		regCode.RemainingUses)

	// Return error, if any
	return err
}
