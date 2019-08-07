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
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/comms/utils"
	"gitlab.com/elixxir/crypto/signature/rsa"
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

	//Set globals for permissioning server
	permissioningCert, err := x509.ParseCertificate(cert)
	if err != nil {
		jww.ERROR.Printf("Failed to parse permissioning server's cert: %+v. Permissioning cert is %+v",
			err, permissioningCert)
	}
	permissioningKey, err := rsa.LoadPrivateKeyFromPem(key)
	if err != nil {
		jww.ERROR.Printf("Failed to parse permissioning server's key: %+v. PermissioningKey is %+v",
			err, permissioningKey)
	}

	// Start the communication server
	registrationImpl.Comms = registration.StartRegistrationServer(params.Address,
		NewRegistrationImpl(), cert, key)

	// Wait forever to prevent process from ending
	select {}
}

// Saves the RSA public key to a JSON file
// and returns registration implementation
func NewRegistrationImpl() *RegistrationImpl {
	return &RegistrationImpl{}
}

// Handle registration attempt by a Client
//TODO: remove the args and returns, removing y,p,q,g, only return the signed public key
func (m *RegistrationImpl) RegisterUser(registrationCode string, rsaPEMBlock []byte) (H, S []byte, err error) {

	// Check database to verify given registration code
	err = database.PermissioningDb.UseCode(registrationCode)
	if err != nil {
		// Invalid registration code, return an error
		jww.ERROR.Printf("Error validating registration code: %s", err)
		return make([]byte, 0), make([]byte, 0), err
	}
	//RSA signature's in PEM format for RSA signature, in which you can apparentally pull the hash from
	//TODO: Change this so that it preps a signature option for privKey.Sign()
	//Concatenate Client public key byte slices

	//Pull the signature option from the pem block
	block, _ := pem.Decode(rsaPEMBlock)
	if block == nil {
		err := fmt.Sprintf("failed to parse PEM block containing the key")
		return nil, nil, errors.New(err)
	}
	//pull the public key from the byte slice..need??
	rsaPubKey, err := x509.ParsePKCS1PublicKey(rsaPEMBlock)
	if err != nil {
		return nil, nil, err
	}

	//What hash to use??
	// Use hardcoded keypair to sign Client-provided public key
	sig, err := privateKey.Sign(rand.Reader, rsaPEMBlock, crypto.BLAKE2b_256)
	if err != nil {
		// Unable to sign public key, return an error
		jww.ERROR.Printf("Error signing client public key: %s", err)
		return make([]byte, 0), make([]byte, 0),
			errors.New("unable to sign client public key")
	}

	// Return signed public key to Client with empty error field
	jww.INFO.Printf("Verification complete for registration code %s",
		registrationCode)
	return data, sig, nil
}
