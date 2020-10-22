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
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/primitives/rateLimiting"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"sync"
	"time"
)

// The main registration instance object
type RegistrationImpl struct {
	Comms                *registration.Comms
	params               *Params
	State                *storage.NetworkState
	Stopped              *uint32
	permissioningCert    *x509.Certificate
	ndfOutputPath        string
	NdfReady             *uint32
	certFromFile         string
	registrationLimiting *rateLimiting.Bucket
	disableGatewayPing   bool

	//registration status trackers
	numRegistered int
	//FIXME: it is possible that polling lock and registration lock
	// do the same job and could conflict. reconsiderations of this logic
	// may be fruitful
	registrationLock  sync.Mutex
	beginScheduling   chan struct{}
	registrationTimes map[id.ID]int64

	NDFLock sync.Mutex
}

//function used to schedule nodes
type SchedulingAlgorithm func(params []byte, state *storage.NetworkState) error

// Configure and start the Permissioning Server
func StartRegistration(params Params) (*RegistrationImpl, error) {

	// Initialize variables
	ndfReady := uint32(0)
	roundCreationStopped := uint32(0)

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

	// Initialize the state tracking object
	state, err := storage.NewState(pk)
	if err != nil {
		return nil, err
	}

	// Build default parameters
	regImpl := &RegistrationImpl{
		State:              state,
		params:             &params,
		ndfOutputPath:      params.NdfOutputPath,
		NdfReady:           &ndfReady,
		Stopped:            &roundCreationStopped,
		numRegistered:      0,
		beginScheduling:    make(chan struct{}, 1),
		disableGatewayPing: params.disableGatewayPing,
		registrationTimes:  make(map[id.ID]int64),
	}

	//regImpl.registrationLimiting = rateLimiting.Create(params.userRegCapacity, params.userRegLeakRate)
	regImpl.registrationLimiting = rateLimiting.CreateBucket(params.userRegCapacity, params.userRegCapacity, params.userRegLeakPeriod, func(u uint32, i int64) {})

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
		for i, n := range def.Nodes {
			ndfNodeID, err := id.Unmarshal(n.ID)
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

		//take the polling lock
		ns.GetPollingLock().Lock()

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
	impl.Functions.RegisterUser = func(regCode string, pubKey string) ([]byte, error) {
		response, err := instance.RegisterUser(regCode, pubKey)
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
	impl.Functions.RegisterNode = func(salt []byte, serverAddr, serverTlsCert, gatewayAddr,
		gatewayTlsCert, registrationCode string) error {

		err := instance.RegisterNode(salt, serverAddr, serverTlsCert, gatewayAddr,
			gatewayTlsCert, registrationCode)
		if err != nil {
			jww.ERROR.Printf("RegisterNode error: %+v", err)
		}

		return err
	}
	impl.Functions.PollNdf = func(theirNdfHash []byte, auth *connect.Auth) ([]byte, error) {

		response, err := instance.PollNdf(theirNdfHash, auth)
		if err != nil && err.Error() != ndf.NO_NDF {
			jww.ERROR.Printf("PollNdf error: %+v", err)
		}

		return response, err
	}

	impl.Functions.Poll = func(msg *pb.PermissioningPoll, auth *connect.Auth, serverAddress string) (*pb.PermissionPollResponse, error) {
		//ensure a bad poll can not take down the permisisoning server
		response, err := instance.Poll(msg, auth, serverAddress)

		return response, err
	}

	// This comm is not authenticated as servers call this early in their
	//lifecycle to check if they've already registered
	impl.Functions.CheckRegistration = func(msg *pb.RegisteredNodeCheck) (confirmation *pb.RegisteredNodeConfirmation, e error) {

		response, e := instance.CheckNodeRegistration(msg)
		// Returning any errors, such as database errors, would result in too much
		// leaked data for a public call.
		return &pb.RegisteredNodeConfirmation{IsRegistered: response}, e

	}

	return impl
}

func (m *RegistrationImpl) GetDisableGatewayPingFlag() bool {
	return m.disableGatewayPing
}
