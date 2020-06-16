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
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
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
	numRegistered int
	//FIXME: it is possible that polling lock and registration lock
	// do the same job and could conflict. reconsiderations of this logic
	// may be fruitful
	registrationLock sync.Mutex
	beginScheduling  chan struct{}
	QuitChans

	NDFLock sync.Mutex
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
	schedulingKillTimeout     time.Duration
	closeTimeout              time.Duration
	minimumNodes              uint32
	udbId                     []byte
	minGatewayVersion         version.Version
	minServerVersion          version.Version
	roundIdPath               string
	updateIdPath              string
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
func StartRegistration(params Params, done chan bool) (*RegistrationImpl, error) {

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
	state, err := storage.NewState(pk, params.roundIdPath, params.updateIdPath)
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

		numRegistered:   0,
		beginScheduling: make(chan struct{}, 1),
	}

	// Create timer and channel to be used by routine that clears the number of
	// registrations every time the ticker activates
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
		// fixme: consider removing. this allows clients to remain agnostic of teaming order
		//  by forcing team order == ndf order for simple non-random
		Nodes:    make([]ndf.Node, 0),
		Gateways: make([]ndf.Gateway, 0),
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
	regImpl.Comms = registration.StartRegistrationServer(&id.Permissioning,
		params.Address, NewImplementation(regImpl),
		[]byte(regImpl.certFromFile), key)

	// In the noTLS pathway, disable authentication
	if noTLS {
		regImpl.Comms.DisableAuth()
	}

	return regImpl, nil
}

// Tracks nodes banned from the network. Sends an update to the scheduler
func BannedNodeTracker(impl *RegistrationImpl) error {
	state := impl.State
	// Search the database for any banned nodes
	bannedNodes, err := storage.PermissioningDb.GetNodesByStatus(node.Banned)
	if err != nil {
		return errors.Errorf("Failed to get nodes by %s status: %v", node.Banned, err)
	}

	impl.NDFLock.Lock()
	defer impl.NDFLock.Unlock()
	def := state.GetFullNdf().Get()

	// Parse through the returned node list
	for _, n := range bannedNodes {
		// Convert the id into an id.ID
		nodeId, err := id.Unmarshal(n.Id)
		if err != nil {
			return errors.Errorf("Failed to convert node %s to id.ID: %v", n.Id, err)
		}

		var newNodes []ndf.Node
		// Loop through NDF nodes to remove any that are banned
		for i, node := range def.Nodes {
			ndfNodeID, err := id.Unmarshal(node.ID)
			if err != nil {
				return errors.WithMessage(err, "Failed to unmarshal node id from NDF")
			}
			if ndfNodeID.Cmp(nodeId) {
				continue
			} else {
				newNodes = append(newNodes, def.Nodes[i])
			}
		}
		if len(newNodes) != len(def.Nodes) {
			def.Nodes = newNodes
			err = state.UpdateNdf(def)
			if err != nil {
				return errors.WithMessage(err, "Failed to update NDF after bans")
			}
		}

		// Get the node from the nodeMap
		ns := state.GetNodeMap().GetNode(nodeId)
		var nun node.UpdateNotification
		// If the node is already banned do not attempt to re-ban
		if ns == nil || ns.IsBanned() {
			continue
		}

		// Ban the node, propagating the ban to the node's state
		nun, err = ns.Ban()
		if err != nil {
			return errors.WithMessage(err, "Could not ban node")
		}

		/// Send the node's update notification to the scheduler
		err = state.SendUpdateNotification(nun)
		if err != nil {
			return errors.WithMessage(err, "Could not send update notification")
		}
	}

	return nil
}

// NewImplementation returns a registration server Handler
func NewImplementation(instance *RegistrationImpl) *registration.Implementation {
	impl := registration.NewImplementation()
	impl.Functions.RegisterUser = func(
		registrationCode, pubKey string) (signature []byte, err error) {

		result := make(chan bool)

		var response []byte

		go func() {
			defer func() {
				if r := recover(); r != nil {
					err = errors.Errorf("Register User crash recovered: %+v", r)
					jww.ERROR.Printf("Register User crash recovered: %+v", r)
					result <- true
				}
			}()

			response, err = instance.RegisterUser(registrationCode, pubKey)
			if err != nil {
				jww.ERROR.Printf("RegisterUser error: %+v", err)
			}
			result <- true
		}()

		<-result

		return response, err
	}

	impl.Functions.GetCurrentClientVersion = func() (version string, err error) {
		result := make(chan bool)

		var response string

		go func() {
			defer func() {
				if r := recover(); r != nil {
					err = errors.Errorf("GetCurrentClientVersion crash recovered: %+v", r)
					jww.ERROR.Printf("GetCurrentClientVersion crash recovered: %+v", r)
					result <- true
				}
			}()

			response, err = instance.GetCurrentClientVersion()
			if err != nil {
				jww.ERROR.Printf("GetCurrentClientVersion error: %+v", err)
			}
			result <- true
		}()

		<-result

		return response, err
	}
	impl.Functions.RegisterNode = func(ID *id.ID, ServerAddr, ServerTlsCert,
		GatewayAddr, GatewayTlsCert, RegistrationCode string) error {

		result := make(chan bool)
		var err error

		go func() {
			defer func() {
				if r := recover(); r != nil {
					err = errors.Errorf("RegisterNode crash recovered: %+v", r)
					jww.ERROR.Printf("RegisterNode crash recovered: %+v", r)
					result <- true
				}
			}()

			err = instance.RegisterNode(ID, ServerAddr,
				ServerTlsCert, GatewayAddr, GatewayTlsCert, RegistrationCode)
			if err != nil {
				jww.ERROR.Printf("RegisterNode error: %+v", err)
			}
			result <- true
		}()

		<-result

		return err
	}
	impl.Functions.PollNdf = func(theirNdfHash []byte, auth *connect.Auth) ([]byte, error) {

		result := make(chan bool)
		var err error
		var response []byte

		go func() {
			defer func() {
				if r := recover(); r != nil {
					err = errors.Errorf("PollNdf crash recovered: %+v", r)
					jww.ERROR.Printf("PollNdf crash recovered: %+v", r)
					result <- true
				}
			}()

			response, err = instance.PollNdf(theirNdfHash, auth)
			if err != nil && err.Error() != ndf.NO_NDF {
				jww.ERROR.Printf("PollNdf error: %+v", err)
			}
			result <- true
		}()

		<-result

		return response, err
	}

	impl.Functions.Poll = func(msg *pb.PermissioningPoll, auth *connect.Auth, serverAddress string) (*pb.PermissionPollResponse, error) {
		//ensure a bad poll can not take down the permisisoning server
		result := make(chan bool)

		response := &pb.PermissionPollResponse{}
		var err error

		go func() {
			defer func() {
				if r := recover(); r != nil {
					err = errors.Errorf("Unified Poll crash recovered: %+v", r)
					jww.ERROR.Printf("Unified Poll crash recovered: %+v", r)
					result <- true
				}
			}()

			response, err = instance.Poll(msg, auth, serverAddress)
			if err != nil && err.Error() != ndf.NO_NDF {
				jww.ERROR.Printf("Unified Poll error: %+v", err)
			}
			result <- true
		}()

		<-result

		return response, err
	}

	// This comm is not authenticated as servers call this early in their
	//lifecycle to check if they've already registered
	impl.Functions.CheckRegistration = func(msg *pb.RegisteredNodeCheck) (confirmation *pb.RegisteredNodeConfirmation, e error) {
		response := instance.CheckNodeRegistration(msg.RegCode)

		// Returning any errors, such as database errors, would result in too much
		// leaked data for a public call.
		return &pb.RegisteredNodeConfirmation{IsRegistered: response}, nil

	}

	return impl
}

func recoverable(f func() error, source string) error {
	result := make(chan bool)
	var err error
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err = errors.Errorf("crash recovered: %+v, %+v", source, r)
				result <- true
			}
		}()
		err = f()
		result <- true
	}()
	<-result
	return err
}
