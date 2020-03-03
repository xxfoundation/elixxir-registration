////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating the impl and params objects for the server

package cmd

import (
	"crypto/x509"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"sync"
	"time"
)

type RegistrationImpl struct {
	Comms                   *registration.Comms
	permissioningCert       *x509.Certificate
	permissioningKey        *rsa.PrivateKey
	ndfOutputPath           string
	nodeCompleted           chan struct{}
	registrationCompleted   chan struct{}
	NumNodesInNet           int
	fullNdfHash             []byte
	partialNdfHash          []byte
	fullNdf                 []byte
	partialNdf              []byte
	ndfLock                 sync.RWMutex
	certFromFile            string
	registrationsRemaining  *uint64
	maxRegistrationAttempts uint64
}

type Params struct {
	Address                   string
	CertPath                  string
	KeyPath                   string
	NdfOutputPath             string
	NsCertPath                string
	NsAddress                 string
	NumNodesInNet             int
	cmix                      ndf.Group
	e2e                       ndf.Group
	publicAddress             string
	maxRegistrationAttempts   uint64
	registrationCountDuration time.Duration
}

// toGroup takes a group represented by a map of string to string
// and uses the prime, small prime and generator to  created
// and returns a an ndf group object.
func toGroup(grp map[string]string) (*ndf.Group, error) {
	jww.DEBUG.Printf("Group is: %v", grp)
	pStr, pOk := grp["prime"]
	gStr, gOk := grp["generator"]

	if !gOk || !pOk {
		return nil, errors.Errorf("Invalid Group Config "+
			"(prime: %v, generator: %v", pOk, gOk)
	}
	return &ndf.Group{Prime: pStr, Generator: gStr}, nil
}

// Configure and start the Permissioning Server
func StartRegistration(params Params) *RegistrationImpl {
	jww.INFO.Printf("Starting registration...")
	regImpl := &RegistrationImpl{}
	var cert, key []byte
	var err error
	regImpl.fullNdfHash = make([]byte, 0)

	// Setup registration limiting variables
	regImpl.maxRegistrationAttempts = params.maxRegistrationAttempts
	maxRegistrationAttempts := uint64(0)
	regImpl.registrationsRemaining = &maxRegistrationAttempts

	// Create timer and channel to be used by routine that clears the number of
	// registrations every time the ticker activates
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(params.registrationCountDuration)
		regImpl.registrationCapacityRestRunner(ticker, done)
	}()

	// Read in private key
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

	}

	regImpl.NumNodesInNet = len(RegistrationCodes)
	regImpl.ndfOutputPath = params.NdfOutputPath

	regHandler := NewImplementation(regImpl)
	// Start the communication server
	regImpl.Comms = registration.StartRegistrationServer(id.PERMISSIONING,
		params.Address,
		regHandler, cert, key)

	//In the noTLS pathway, disable authentication
	if noTLS {
		regImpl.Comms.DisableAuth()
	}

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
	impl.Functions.RegisterNode = func(ID []byte, ServerAddr, ServerTlsCert,
		GatewayAddr, GatewayTlsCert, RegistrationCode string) error {
		return instance.RegisterNode(ID, ServerAddr,
			ServerTlsCert, GatewayAddr, GatewayTlsCert,
			RegistrationCode)
	}
	impl.Functions.PollNdf = func(theirNdfHash []byte, auth *connect.Auth) ([]byte, error) {
		return instance.PollNdf(theirNdfHash, auth)
	}

	return impl
}
