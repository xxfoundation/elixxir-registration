////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating client registration callbacks for hooking into comms library

package cmd

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/registration/storage"
	"sync/atomic"
	"time"
)

const (
	defaultMaxRegistrationAttempts   = uint64(500)
	defaultRegistrationCountDuration = time.Hour * 24
)

// Handle registration attempt by a Client
func (m *RegistrationImpl) RegisterUser(registrationCode, pubKey string) (
	signature []byte, err error) {

	// Check for pre-existing registration for this public key
	if user, err := storage.PermissioningDb.GetUser(pubKey); err == nil && user != nil {
		jww.INFO.Printf("Previous registration found for %s", pubKey)
	} else {
		// Check database to verify given registration code
		jww.INFO.Printf("Attempting to use registration code %+v...",
			registrationCode)
		err = storage.PermissioningDb.UseCode(registrationCode)
		if err != nil {
			// Check if the max registration attempts have been reached
			if atomic.LoadUint64(m.registrationsRemaining) >= m.maxRegistrationAttempts {
				// Invalid registration code, return an error
				return make([]byte, 0), errors.Errorf(
					"Error validating registration code: %+v", err)
			} else {
				atomic.AddUint64(m.registrationsRemaining, 1)
				jww.INFO.Printf("Incremented registration counter to %+v (max %v)",
					atomic.LoadUint64(m.registrationsRemaining), m.maxRegistrationAttempts)
			}
		}

		// Record the user public key for duplicate registration support
		err = storage.PermissioningDb.InsertUser(pubKey)
		if err != nil {
			jww.WARN.Printf("Unable to store user: %+v",
				errors.New(err.Error()))
		}
	}

	// Use hardcoded keypair to sign Client-provided public key
	//Create a hash, hash the pubKey and then truncate it
	h := sha256.New()
	h.Write([]byte(pubKey))
	data := h.Sum(nil)
	sig, err := rsa.Sign(rand.Reader, m.permissioningKey, crypto.SHA256, data, nil)
	if err != nil {
		return make([]byte, 0), errors.Errorf(
			"Unable to sign client public key: %+v", err)
	}

	// Return signed public key to Client
	jww.INFO.Printf("Registration for code %+v complete!", registrationCode)
	return sig, nil
}

// This has to be part of RegistrationImpl and has to return an error because
// of the way our comms are structured
func (m *RegistrationImpl) GetCurrentClientVersion() (version string, err error) {
	clientVersionLock.RLock()
	defer clientVersionLock.RUnlock()
	return clientVersion, nil
}

// registrationCapacityRestRunner sets the registrations remaining to zero when
// a ticker occurs.
func (m *RegistrationImpl) registrationCapacityRestRunner(ticker *time.Ticker, done chan bool) {
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			atomic.StoreUint64(m.registrationsRemaining, 0)
		}
	}
}
