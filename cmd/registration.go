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
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"time"
)

const (
	defaultMaxRegistrationAttempts   = uint64(500)
	defaultRegistrationCountDuration = time.Hour * 24
)

var rateLimitErr = errors.New("Clients have exceeded registration rate limit")

// Handle registration attempt by a Client
func (m *RegistrationImpl) RegisterUser(pubKey string) (
	signature []byte, err error) {
	// Check for pre-existing registration for this public key first
	if user, err := storage.PermissioningDb.GetUser(pubKey); err == nil && user != nil {
		jww.INFO.Printf("Previous registration found for %s", pubKey)
	}

	// Check rate limiting
	if !m.registrationLimiting.Add(1) {
		// Rate limited, fail early
		// Will logging result in problems in case of ddos attempt?
		return nil, rateLimitErr
	}

	// Use hardcoded keypair to sign Client-provided public key
	//Create a hash, hash the pubKey and then truncate it
	h := sha256.New()
	h.Write([]byte(pubKey))
	data := h.Sum(nil)
	sig, err := rsa.Sign(rand.Reader, m.State.GetPrivateKey(), crypto.SHA256, data, nil)
	if err != nil {
		return make([]byte, 0), errors.Errorf(
			"Unable to sign client public key: %+v", err)
	}

	// Record the user public key for duplicate registration support
	err = storage.PermissioningDb.InsertUser(pubKey)
	if err != nil {
		jww.WARN.Printf("Unable to store user: %+v",
			errors.New(err.Error()))
	}

	// Return signed public key to Client
	jww.INFO.Printf("Registration for public key %+v complete!", pubKey)
	return sig, nil
}

// This has to be part of RegistrationImpl and has to return an error because
// of the way our comms are structured
func (m *RegistrationImpl) GetCurrentClientVersion() (version string, err error) {
	clientVersionLock.RLock()
	defer clientVersionLock.RUnlock()
	return clientVersion, nil
}
