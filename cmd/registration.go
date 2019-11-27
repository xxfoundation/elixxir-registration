////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating client registration callbacks for hooking into comms library

package cmd

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/database"
	"sync"
	"time"
)

type RegistrationImpl struct {
	Comms                 *registration.Comms
	permissioningCert     *x509.Certificate
	permissioningKey      *rsa.PrivateKey
	ndfOutputPath         string
	nodeCompleted         chan struct{}
	registrationCompleted chan struct{}
	NumNodesInNet         int
	ndfLock               sync.RWMutex
	regNdfHash            []byte
	certFromFile          string
	ndfJson               []byte
}

type Params struct {
	Address       string
	CertPath      string
	KeyPath       string
	NdfOutputPath string
	NumNodesInNet int
	cmix          ndf.Group
	e2e           ndf.Group
	publicAddress string
}

type connectionID string

func (c connectionID) String() string {
	return (string)(c)
}

// toGroup takes a group represented by a map of string to string
// and uses the prime, small prime and generator to  created
// and returns a an ndf group object.
func toGroup(grp map[string]string) ndf.Group {
	jww.DEBUG.Printf("group is: %v", grp)
	pStr, pOk := grp["prime"]
	gStr, gOk := grp["generator"]

	if !gOk || !pOk {
		jww.FATAL.Panicf("Invalid Group Config "+
			"(prime: %v, generator: %v",
			pOk, gOk)
	}

	return ndf.Group{Prime: pStr, Generator: gStr}

}

// Configure and start the Permissioning Server
func StartRegistration(params Params) *RegistrationImpl {
	jww.DEBUG.Printf("Starting registration\n")
	regImpl := &RegistrationImpl{}
	var cert, key []byte
	var err error

	regImpl.regNdfHash = make([]byte, 0)
	if !noTLS {
		// Read in TLS keys from files
		cert, err = utils.ReadFile(params.CertPath)
		if err != nil {
			jww.ERROR.Printf("failed to read certificate at %+v: %+v", params.CertPath, err)
		}
		// Set globals for permissioning server
		regImpl.certFromFile = string(cert)
		regImpl.permissioningCert, err = tls.LoadCertificate(string(cert))
		if err != nil {
			jww.ERROR.Printf("Failed to parse permissioning server cert: %+v. "+
				"Permissioning cert is %+v",
				err, regImpl.permissioningCert)
		}
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

		jww.DEBUG.Printf("permissioningCert: %+v\n", regImpl.permissioningCert)
		jww.DEBUG.Printf("permissioning public key: %+v\n", regImpl.permissioningCert.PublicKey)
		jww.DEBUG.Printf("permissioning private key: %+v\n", regImpl.permissioningKey)
	}
	regImpl.NumNodesInNet = len(RegistrationCodes)
	regImpl.ndfOutputPath = params.NdfOutputPath

	regHandler := NewImplementation(regImpl)

	// Start the communication server
	regImpl.Comms = registration.StartRegistrationServer(params.Address,
		regHandler, cert, key)

	regImpl.nodeCompleted = make(chan struct{}, regImpl.NumNodesInNet)
	regImpl.registrationCompleted = make(chan struct{}, 1)
	return regImpl
}

// NewImplementation returns a registertation server Handler
func NewImplementation(instance *RegistrationImpl) *registration.Implementation {
	impl := registration.NewImplementation()
	impl.Functions.RegisterUser = func(registrationCode, pubKey string) (
		signature []byte, err error) {
		return instance.RegisterUser(registrationCode, pubKey)
	}
	impl.Functions.GetCurrentClientVersion = func() (version string,
		err error) {
		return instance.GetCurrentClientVersion()
	}
	impl.Functions.RegisterNode = func(ID []byte, ServerTlsCert,
		GatewayAddr, GatewayTlsCert, RegistrationCode string) error {
		return instance.RegisterNode(ID,
			ServerTlsCert, GatewayAddr, GatewayTlsCert,
			RegistrationCode)
	}
	impl.Functions.PollNdf = func(theirNdfHash []byte) ([]byte, error) {
		return instance.PollNdf(theirNdfHash)
	}

	return impl
}

// Handle registration attempt by a Client
func (m *RegistrationImpl) RegisterUser(registrationCode, pubKey string) (
	signature []byte, err error) {
	jww.INFO.Printf("Verifying for registration code %+v",
		registrationCode)
	// Check database to verify given registration code
	err = database.PermissioningDb.UseCode(registrationCode)
	if err != nil {
		// Invalid registration code, return an error
		errMsg := errors.Errorf(
			"Error validating registration code: %+v", err)
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
		errMsg := errors.Errorf(
			"unable to sign client public key: %+v", err)
		jww.ERROR.Printf("%+v", errMsg)
		return make([]byte, 0),
			err
	}

	jww.INFO.Printf("Verification complete for registration code %+v",
		registrationCode)
	// Return signed public key to Client with empty error field
	return sig, nil
}

//PollNdf handles the client polling for an updated NDF
func (m *RegistrationImpl) PollNdf(theirNdfHash []byte) ([]byte, error) {
	//If permissioning is enabled, check the permissioning's hash against the client's ndf
	if !disablePermissioning {
		if theirNdfHash == nil {
			return m.serverNdfRequest()
		}

		return m.clientNdfRequest(theirNdfHash)
	}
	jww.DEBUG.Printf("Permissioning disabled, telling requester that their ndf is up-to-date")
	//If permissioning is disabled, inform the client that it has the correct ndf
	return nil, nil

}

//clientNdfRequest handles when a client requests an ndf from permissioning
func (m *RegistrationImpl) clientNdfRequest(theirNdfHash []byte) ([]byte, error) {
	// Lock the reading of regNdfHash and check if it's been writen to
	m.ndfLock.RLock()
	ndfHashLen := len(m.regNdfHash)
	m.ndfLock.RUnlock()
	//Check that the registration server has built an NDF
	if ndfHashLen == 0 {
		errMsg := errors.Errorf("Permissioning server does not have an ndf to give to client")
		jww.WARN.Printf(errMsg.Error())
		return nil, errMsg
	}

	//If both the sender's ndf hash and the permissioning NDF hash match
	//  no need to pass anything through the comm
	if bytes.Compare(m.regNdfHash, theirNdfHash) == 0 {
		return nil, nil
	}

	jww.DEBUG.Printf("Returning a new NDF to client!")
	//Send the json of the ndf
	return m.ndfJson, nil
}

func (m *RegistrationImpl) serverNdfRequest() ([]byte, error) {
	timeOut := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-m.registrationCompleted:
			return m.ndfJson, nil
		case <-timeOut.C:
			return nil, errors.Errorf("Permissioning does not have an ndf to give to node server")
		}
	}
}

// This has to be part of RegistrationImpl and has to return an error because
// of the way our comms are structured
func (m *RegistrationImpl) GetCurrentClientVersion() (version string, err error) {
	clientVersionLock.RLock()
	defer clientVersionLock.RUnlock()
	return clientVersion, nil
}
