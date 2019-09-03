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
	"crypto/x509"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/database"
	"golang.org/x/tools/go/ssa/interp/testdata/src/errors"
)

type RegistrationImpl struct {
	Comms             *registration.RegistrationComms
	permissioningCert *x509.Certificate
	permissioningKey  *rsa.PrivateKey
	ndfOutputPath     string
	completedNodes    chan struct{}
	NumNodesInNet     int
}

type Params struct {
	Address       string
	CertPath      string
	KeyPath       string
	NdfOutputPath string
	NumNodesInNet int
}

type connectionID string

func (c connectionID) String() string {
	return (string)(c)
}

// Configure and start the Permissioning Server
func StartRegistration(params Params) *RegistrationImpl {
	jww.DEBUG.Printf("Starting registration\n")
	regImpl := &RegistrationImpl{}

	var cert, key []byte
	var err error

	if !noTLS {
		// Read in TLS keys from files
		cert, err = utils.ReadFile(params.CertPath)
		if err != nil {
			jww.ERROR.Printf("failed to read certificate at %+v: %+v", params.CertPath, err)
		}

		// Set globals for permissioning server
		regImpl.permissioningCert, err = tls.LoadCertificate(string(cert))
		if err != nil {
			jww.ERROR.Printf("Failed to parse permissioning server cert: %+v. "+
				"Permissioning cert is %+v",
				err, regImpl.permissioningCert)
		}
		jww.DEBUG.Printf("permissioningCert: %+v\n", regImpl.permissioningCert)
		jww.DEBUG.Printf("permissioning public key: %+v\n", regImpl.permissioningCert.PublicKey)
		jww.DEBUG.Printf("permissioning private key: %+v\n", regImpl.permissioningKey)
	}
	regImpl.NumNodesInNet = len(RegistrationCodes)
	key, err = utils.ReadFile(params.KeyPath)
	if err != nil {
		jww.ERROR.Printf("failed to read key at %+v: %+v", params.KeyPath, err)
	}
	regImpl.permissioningKey, err = rsa.LoadPrivateKeyFromPem(key)
	if err != nil {
		jww.ERROR.Printf("Failed to parse permissioning server key: %+v. "+
			"PermissioningKey is %+v",
			err, regImpl.permissioningKey)
	}

	regImpl.ndfOutputPath = params.NdfOutputPath

	// Start the communication server
	regImpl.Comms = registration.StartRegistrationServer(params.Address,
		regImpl, cert, key)

	//TODO: change the buffer length to that set in params..also set in params :)
	regImpl.completedNodes = make(chan struct{}, regImpl.NumNodesInNet)
	return regImpl
}

// Handle registration attempt by a Client
func (m *RegistrationImpl) RegisterUser(registrationCode, pubKey string) (signature []byte, err error) {
	jww.INFO.Printf("Verifying for registration code %+v",
		registrationCode)
	// Check database to verify given registration code
	err = database.PermissioningDb.UseCode(registrationCode)
	if err != nil {
		// Invalid registration code, return an error
		errMsg := errors.New(fmt.Sprintf(
			"Error validating registration code: %+v", err))
		jww.ERROR.Printf("%+v", errMsg)
		return make([]byte, 0), errMsg
	}

	sha := crypto.SHA256

	// Use hardcoded keypair to sign Client-provided public key
	//Create a hash, hash the pubKey and then truncate it
	h := sha256.New()
	h.Write([]byte(pubKey))
	data := h.Sum(nil)
	sig, err := rsa.Sign(rand.Reader, m.permissioningKey, sha, data, nil)
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"unable to sign client public key: %+v", err))
		jww.ERROR.Printf("%+v", errMsg)
		return make([]byte, 0),
			err
	}

	jww.INFO.Printf("Verification complete for registration code %+v",
		registrationCode)
	// Return signed public key to Client with empty error field
	return sig, nil
}
