////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating callbacks for hooking into comms library

package cmd

import (
	"gitlab.com/elixxir/comms/registration"
)

type RegistrationImpl struct{}

// Initializes a Registration Handler interface
func NewRegistrationImpl() registration.Handler {
	return registration.Handler(&RegistrationImpl{})
}

func (m *RegistrationImpl) RegisterUser(registrationCode string, Y, P, Q,
	G []byte) (hash, R, S []byte, err error) {

	// Check registration code database to verify given registration code
	// exists and has at least one remaining use

	// If valid registration code:
	//     Decrement registration code counter in registration code database
	//     Use hardcoded RegistrationServer keypair to sign Client-provided public key
	//     Return signed public key to Client with empty error field

	// If invalid registration code:
	//     Return empty message with relevant error field
	return make([]byte, 0), make([]byte, 0), make([]byte, 0), nil
}
