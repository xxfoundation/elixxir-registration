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
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/storage"
	"sync"
	"time"
)

//generally large buffer, should be roughly as many nodes as are expected
const nodeCompletionChanLen = 1000

// The main registration instance object
type RegistrationImpl struct {
	Comms                   *registration.Comms
	params                  *Params
	State                   *storage.NetworkState
	permissioningCert       *x509.Certificate
	ndfOutputPath           string
	NdfReady                *uint32
	certFromFile            string
	registrationsRemaining  *uint64
	maxRegistrationAttempts uint64

	//registration status trackers
	numRegistered 			int
	registrationLock		sync.Mutex
	beginScheduling			chan struct{}
}

//function used to schedule nodes
type SchedulingAlgorithm func(params []byte, state *storage.NetworkState) error

// Params object for reading in configuration data
type Params struct {
	Address                   string
	CertPath                  string
	KeyPath                   string
	NdfOutputPath             string
	NsCertPath                string
	NsAddress                 string
	cmix                      ndf.Group
	e2e                       ndf.Group
	publicAddress             string
	maxRegistrationAttempts   uint64
	registrationCountDuration time.Duration
	minimumNodes              uint32
	udbId                     []byte
}

// toGroup takes a group represented by a map of string to string,
// then uses the prime and generator to create an ndf group object.
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
func StartRegistration(params Params) (*RegistrationImpl, error) {

	// Initialize variables
	regRemaining := uint64(0)
	ndfReady := uint32(0)

	// Read in private key
	key, err := utils.ReadFile(params.KeyPath)
	if err != nil {
		return nil, errors.Errorf("failed to read key at %+v: %+v",
			params.KeyPath, err)
	}

	pk, err := rsa.LoadPrivateKeyFromPem(key)
	if err != nil {
		return nil, errors.Errorf("Failed to parse permissioning server key: %+v. "+
			"PermissioningKey is %+v", err, pk)
	}

	//initilize the state tracking object
	state, err := storage.NewState(pk)
	if err != nil {
		return nil, err
	}

	// Build default parameters
	regImpl := &RegistrationImpl{
		State:                   state,
		params:                  &params,
		maxRegistrationAttempts: params.maxRegistrationAttempts,
		registrationsRemaining:  &regRemaining,
		ndfOutputPath:           params.NdfOutputPath,
		NdfReady:                &ndfReady,

		numRegistered: 0,
		beginScheduling: make(chan struct{}),
	}

	// Create timer and channel to be used by routine that clears the number of
	// registrations every time the ticker activates
	done := make(chan bool)
	go func() {
		ticker := time.NewTicker(params.registrationCountDuration)
		regImpl.registrationCapacityRestRunner(ticker, done)
	}()

	if !noTLS {
		// Read in TLS keys from files
		cert, err := utils.ReadFile(params.CertPath)
		if err != nil {
			return nil, errors.Errorf("failed to read certificate at %+v: %+v", params.CertPath, err)
		}

		// Set globals for permissioning server
		regImpl.certFromFile = string(cert)
		regImpl.permissioningCert, err = tls.LoadCertificate(string(cert))
		if err != nil {
			return nil, errors.Errorf("Failed to parse permissioning server cert: %+v. "+
				"Permissioning cert is %+v", err, regImpl.permissioningCert)
		}
	}

	// Construct the NDF
	networkDef := &ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:        RegParams.publicAddress,
			TlsCertificate: regImpl.certFromFile,
		},
		Timestamp: time.Now(),
		UDB:       ndf.UDB{ID: RegParams.udbId},
		E2E:       RegParams.e2e,
		CMIX:      RegParams.cmix,
	}

	// Assemble notification server information if configured
	if RegParams.NsCertPath != "" && RegParams.NsAddress != "" {
		nsCert, err := utils.ReadFile(RegParams.NsCertPath)
		if err != nil {
			return nil, errors.Errorf("unable to read notification certificate")
		}
		networkDef.Notification = ndf.Notification{
			Address:        RegParams.NsAddress,
			TlsCertificate: string(nsCert),
		}
	} else {
		jww.WARN.Printf("Configured to run without notifications bot!")
	}

	// update the internal state with the newly-formed NDF
	err = regImpl.State.UpdateNdf(networkDef)
	if err != nil {
		return nil, err
	}

	// Start the communication server
	regImpl.Comms = registration.StartRegistrationServer(id.PERMISSIONING,
		params.Address, NewImplementation(regImpl),
		[]byte(regImpl.certFromFile), key)

	// In the noTLS pathway, disable authentication
	if noTLS {
		regImpl.Comms.DisableAuth()
	}

	return regImpl, nil
}

// NewImplementation returns a registration server Handler
func NewImplementation(instance *RegistrationImpl) *registration.Implementation {
	impl := registration.NewImplementation()
	impl.Functions.RegisterUser = func(
		registrationCode, pubKey string) (signature []byte, err error) {

		response, err := instance.RegisterUser(registrationCode, pubKey)
		if err != nil {
			jww.ERROR.Printf("RegisterUser error: %+v", err)
		}

		return response, err
	}
	impl.Functions.GetCurrentClientVersion = func() (version string, err error) {

		response, err := instance.GetCurrentClientVersion()
		if err != nil {
			jww.ERROR.Printf("GetCurrentClientVersion error: %+v", err)
		}

		return response, err
	}
	impl.Functions.RegisterNode = func(ID []byte, ServerAddr, ServerTlsCert,
		GatewayAddr, GatewayTlsCert, RegistrationCode string) error {

		err := instance.RegisterNode(ID, ServerAddr,
			ServerTlsCert, GatewayAddr, GatewayTlsCert, RegistrationCode)
		if err != nil {
			jww.ERROR.Printf("RegisterNode error: %+v", err)
		}

		return err
	}
	impl.Functions.PollNdf = func(theirNdfHash []byte, auth *connect.Auth) ([]byte, error) {

		response, err := instance.PollNdf(theirNdfHash, auth)
		if err != nil {
			jww.ERROR.Printf("PollNdf error: %+v", err)
		}

		return response, err
	}

	impl.Functions.Poll = func(msg *pb.PermissioningPoll, auth *connect.Auth) (*pb.PermissionPollResponse, error) {

		response, err := instance.Poll(msg, auth)
		if err != nil {
			jww.ERROR.Printf("Poll error: %+v", err)
		}

		return response, err
	}

	return impl
}
