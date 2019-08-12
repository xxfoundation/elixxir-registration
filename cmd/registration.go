////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating client registration callbacks for hooking into comms library

package cmd

import (
	"crypto/rand"
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
	fmt.Println(params.KeyPath)
	cert, err := ioutil.ReadFile(utils.GetFullPath(params.CertPath))
	if err != nil {
		jww.ERROR.Printf("failed to read certificate at %s: %+v", params.CertPath, err)
	}
	fmt.Printf("cert is %+v", cert)
	key, err := ioutil.ReadFile(utils.GetFullPath(params.KeyPath))
	if err != nil {
		jww.ERROR.Printf("failed to read key at %s: %+v", params.KeyPath, err)
	}
	//Set globals for permissioning server
	fmt.Println(cert)

	fmt.Println("this is key")
	fmt.Println(string(key))
	/**tmpKey, err := tls.LoadRSAPrivateKey(string(key))
	if err != nil {
		jww.ERROR.Printf("Failed to parse permissioning server's key: %+v",
			err)
	}/**/

	//pemDecodedKey, rest := pem.Decode(key)
	//TODO figure out how to handle the rest error
	//if len(rest) != 0 {
		//jww.ERROR.Printf("Rest is not nil: %+v", rest)
	//}

	// Start the communication server
	//Make the changes for download topology, now have to return the signed message as well...
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
//TODO: remove the args and returns, removing y,p,q,g, only return the signed public key
func (m *RegistrationImpl) RegisterUser(registrationCode, pubKey string) (signature []byte, err error) {

	// Check database to verify given registration code
	err = database.PermissioningDb.UseCode(registrationCode)
	if err != nil {
		// Invalid registration code, return an error
		jww.ERROR.Printf("Error validating registration code: %s", err)
		return make([]byte, 0), err
	}
	//RSA signature's in PEM format for RSA signature, in which you can apparentally pull the hash from
	//TODO: Change this so that it preps a signature option for privKey.Sign()

	//Reviewer: What hash to use?? is the crypto newDefaultoptions correct??
	// Use hardcoded keypair to sign Client-provided public key
	sig, err := privateKey.Sign(rand.Reader, []byte(pubKey), rsa.NewDefaultOptions())
	if err != nil {
		return make([]byte, 0),
			errors.New("unable to sign client public key")
	}
	//Reviewer: thoughts on keeping this?
	// Return signed public key to Client with empty error field
	jww.INFO.Printf("Verification complete for registration code %s",
		registrationCode)
	return sig, nil
}
