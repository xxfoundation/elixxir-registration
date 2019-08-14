////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating client registration callbacks for hooking into comms library

package cmd

import (
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/comms/utils"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/registration/database"
	"io/ioutil"
)

// Registration Implementation
var registrationImpl RegistrationImpl

// Hardcoded RSA keypair for registration server
var privateKey *rsa.PrivateKey

type RegistrationImpl struct {
	Comms *registration.RegistrationComms
}

type Params struct {
	Address       string
	CertPath      string
	KeyPath       string
	NdfOutputPath string
}

type connectionID string

func (c connectionID) String() string {
	return (string)(c)
}

// Configure and start the Permissioning Server
func StartRegistration(params Params) {

	// Read in TLS keys from files
	cert, err := ioutil.ReadFile(utils.GetFullPath(params.CertPath))
	if err != nil {
		jww.ERROR.Printf("failed to read certificate at %s: %+v", params.CertPath, err)
	}
	key, err := ioutil.ReadFile(utils.GetFullPath(params.KeyPath))
	if err != nil {
		jww.ERROR.Printf("failed to read key at %s: %+v", params.KeyPath, err)
	}

	// Set globals for permissioning server
	permissioningCert, err = tls.LoadCertificate(string(cert))
	if err != nil {
		jww.ERROR.Printf("Failed to parse permissioning server cert: %+v. "+
			"Permissioning cert is %+v",
			err, permissioningCert)
	}
	permissioningKey, err = tls.LoadRSAPrivateKey(string(key))
	if err != nil {
		jww.ERROR.Printf("Failed to parse permissioning server key: %+v. "+
			"PermissioningKey is %+v",
			err, permissioningKey)
	}
	// Start the communication server
	//NOTE: see setPrviateKey
	registrationImpl.Comms = registration.StartRegistrationServer(params.Address,
		&registrationImpl, cert, key)

	// Wait forever to prevent process from ending
	select {}
}

// Saves the RSA public key to a JSON file
// and returns registration implementation
func NewRegistrationImpl() *RegistrationImpl {
	return &RegistrationImpl{}
}

// Handle registration attempt by a Client
func (m *RegistrationImpl) RegisterUser(registrationCode, pubKey string) (signature []byte, err error) {
	jww.INFO.Printf("Verifying for registration code %s",
		registrationCode)
	// Check database to verify given registration code
	err = database.PermissioningDb.UseCode(registrationCode)
	if err != nil {
		// Invalid registration code, return an error
		jww.ERROR.Printf("Error validating registration code: %s", err)
		return make([]byte, 0), err
	}

	signingKey := &rsa.PrivateKey{*permissioningKey}
	// Use hardcoded keypair to sign Client-provided public key
	//Create a hash, hash the pubKey and then truncate it
	hash := sha256.New()
	hashed := hash.Sum([]byte(pubKey))
	hashed = hashed[len(pubKey):]
	sig, err := rsa.Sign(rand.Reader, signingKey, crypto.SHA256, hashed[:], rsa.NewDefaultOptions())
	if err != nil {
		jww.ERROR.Printf("unable to sign client public key: %+v", err)
		return make([]byte, 0),
			err
	}
	//Reviewer: thoughts on keeping this?
	// Return signed public key to Client with empty error field
	jww.INFO.Printf("Verification complete for registration code %s",
		registrationCode)
	return sig, nil
}
