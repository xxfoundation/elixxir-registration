////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating callbacks for hooking into comms library

package cmd

import (
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/registration/database"
)

type RegistrationImpl struct{}

// Initializes a Registration Handler interface
func NewRegistrationImpl() registration.Handler {
	return registration.Handler(&RegistrationImpl{})
}

// Handle registration attempt by a Client
func (m *RegistrationImpl) RegisterUser(registrationCode string, Y, P, Q,
	G []byte) (hash, R, S []byte, err error) {

	// Check registration code database to verify given registration code
	err = database.RegCodes.UseCode(registrationCode)
	if err != nil {
		// Invalid registration code, return an error
		return make([]byte, 0), make([]byte, 0), make([]byte, 0), err
	}

	// TODO: Use hardcoded RegistrationServer keypair to sign Client-provided public key
	// TODO: Return signed public key to Client with empty error field
	return make([]byte, 0), make([]byte, 0), make([]byte, 0), nil
}
