////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating the impl and params objects for the server

package cmd

import (
	"crypto/x509"
	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/netTime"
	"gitlab.com/xx_network/primitives/utils"
	"sync"
	"time"
)

// The main registration instance object
type RegistrationImpl struct {
	Comms              *registration.Comms
	params             *Params
	State              *storage.NetworkState
	Stopped            *uint32
	permissioningCert  *x509.Certificate
	ndfOutputPath      string
	NdfReady           *uint32
	certFromFile       string
	disableGatewayPing bool

	// registration status trackers
	numRegistered int
	// FIXME: it is possible that polling lock and registration lock
	// do the same job and could conflict. reconsiderations of this logic
	// may be fruitful
	registrationLock sync.Mutex
	beginScheduling  chan struct{}
	// TODO-kill this
	registrationTimes map[id.ID]int64

	// GeoLite2 database reader instance for getting info about an IP address
	geoIPDB *geoip2.Reader

	// Status of the geoip2.Reader; signals if the reader is running or stopped
	geoIPDBStatus geoipStatus

	NDFLock sync.Mutex
}

// function used to schedule nodes
type SchedulingAlgorithm func(params []byte, state *storage.NetworkState) error

// Configure and start the Permissioning Server
func StartRegistration(params Params) (*RegistrationImpl, error) {

	// Initialize variables
	ndfReady := uint32(0)
	roundCreationStopped := uint32(0)

	// Read in private key
	rsaKeyPem, err := utils.ReadFile(params.KeyPath)
	if err != nil {
		return nil, errors.Errorf("failed to read key at %+v: %+v",
			params.KeyPath, err)
	}

	rsaPrivateKey, err := rsa.LoadPrivateKeyFromPem(rsaKeyPem)
	if err != nil {
		return nil, errors.Errorf("Failed to parse permissioning server key: %+v. "+
			"PermissioningKey is %+v", err, rsaPrivateKey)
	}

	// Check if any address space sizes are saved to the database and if not,
	// use the size from the config file
	if _, err := storage.PermissioningDb.GetLatestEphemeralLength(); err != nil {
		jww.WARN.Printf("Using address space size of %d from config due to "+
			"error receiving address space size from storage: %s",
			params.addressSpaceSize, err)

		err := storage.PermissioningDb.InsertEphemeralLength(
			&storage.EphemeralLength{
				Length:    params.addressSpaceSize,
				Timestamp: netTime.Now(),
			})
		if err != nil {
			return nil, errors.Errorf("Failed to save initial address space "+
				"size %d to database: %+v", params.addressSpaceSize, err)
		}
	}

	// Get list of addresses spaces from database
	addressSpaces, newestAddressSpace, err := GetAddressSpaceSizesFromStorage(
		storage.PermissioningDb)
	if err != nil {
		return nil, errors.Errorf("Failed to get ephemeral ID lengths from "+
			"database: %v.", err)
	}

	// Initialize the state tracking object
	state, err := storage.NewState(
		rsaPrivateKey, uint32(newestAddressSpace.Size), params.NdfOutputPath)
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

	// Load the UDB cert from file
	udbCert, err := utils.ReadFile(params.udbCertPath)
	if err != nil {
		return nil, errors.Errorf("failed to read UDB cert: %+v", err)
	}

	// Construct the NDF
	networkDef := &ndf.NetworkDefinition{
		Registration: ndf.Registration{
			Address:                   RegParams.publicAddress,
			TlsCertificate:            regImpl.certFromFile,
			EllipticPubKey:            state.GetEllipticPublicKey().MarshalText(),
			ClientRegistrationAddress: RegParams.clientRegistrationAddress,
		},
		Timestamp: time.Now(),
		UDB: ndf.UDB{
			ID:       RegParams.udbId,
			Cert:     string(udbCert),
			Address:  RegParams.udbAddress,
			DhPubKey: RegParams.udbDhPubKey,
		},
		E2E:  RegParams.e2e,
		CMIX: RegParams.cmix,
		// fixme: consider removing. this allows clients to remain agnostic of teaming order
		//  by forcing team order == ndf order for simple non-random
		Nodes:         make([]ndf.Node, 0),
		Gateways:      make([]ndf.Gateway, 0),
		AddressSpace:  addressSpaces,
		ClientVersion: RegParams.minClientVersion.String(),
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
		[]byte(regImpl.certFromFile), rsaKeyPem)

	// In the noTLS pathway, disable authentication
	if noTLS {
		regImpl.Comms.DisableAuth()
	}

	// If the the GeoIP2 database file is supplied, then use it to open the
	// GeoIP2 reader; otherwise, error if randomGeoBinning is not set
	if params.geoIPDBFile != "" {
		regImpl.geoIPDB, err = geoip2.Open(params.geoIPDBFile)
		if err != nil {
			return nil,
				errors.Errorf("failed to load GeoIP2 database file: %+v", err)
		}

		// Set the GeoIP2 reader to running
		regImpl.geoIPDBStatus.ToRunning()
	} else if !params.randomGeoBinning {
		jww.FATAL.Panic("Must provide either a MaxMind GeoLite2 compatible " +
			"database file or set the 'randomGeoBinning' flag.")
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
	def := state.GetUnprunedNdf()

	// Parse through the returned node list
	for _, n := range bannedNodes {
		// Convert the id into an id.ID
		nodeId, err := id.Unmarshal(n.Id)
		if err != nil {
			return errors.Errorf("Failed to convert node %s to id.ID: %v", n.Id, err)
		}

		gatewayID := nodeId.DeepCopy()
		gatewayID.SetType(id.Gateway)

		var remainingNodes []ndf.Node
		var remainingGateways []ndf.Gateway
		// Loop through NDF nodes to remove any that are banned
		for i, n := range def.Nodes {
			ndfNodeID, err := id.Unmarshal(n.ID)
			if err != nil {
				return errors.WithMessage(err, "Failed to unmarshal node id from NDF")
			}
			if ndfNodeID.Cmp(nodeId) {
				continue
			} else {
				remainingNodes = append(remainingNodes, def.Nodes[i])
			}
		}

		for i, g := range def.Gateways {
			ndfGatewayID, err := id.Unmarshal(g.ID)
			if err != nil {
				return errors.WithMessage(err, "Failed to unmarshal gateway id from NDF")
			}
			if ndfGatewayID.Cmp(gatewayID) {
				continue
			} else {
				remainingGateways = append(remainingGateways, def.Gateways[i])
			}
		}

		update := false

		if len(remainingNodes) != len(def.Nodes) {
			def.Nodes = remainingNodes
			update = true
		}

		if len(remainingGateways) != len(def.Gateways) {
			def.Gateways = remainingGateways
			update = true
		}

		if update {
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

		// take the polling lock
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
	impl.Functions.RegisterNode = func(salt []byte, serverAddr, serverTlsCert, gatewayAddr,
		gatewayTlsCert, registrationCode string) error {

		err := instance.RegisterNode(salt, serverAddr, serverTlsCert, gatewayAddr,
			gatewayTlsCert, registrationCode)
		if err != nil {
			jww.ERROR.Printf("RegisterNode error: %+v", err)
		}

		return err
	}
	impl.Functions.PollNdf = func(theirNdfHash []byte) (*pb.NDF, error) {

		response, err := instance.PollNdf(theirNdfHash)
		if err != nil && err.Error() != ndf.NO_NDF {
			jww.ERROR.Printf("PollNdf error: %+v", err)
		}

		return response, err
	}

	impl.Functions.Poll = func(msg *pb.PermissioningPoll, auth *connect.Auth) (*pb.PermissionPollResponse, error) {
		// ensure a bad poll can not take down the permisisoning server
		response, err := instance.Poll(msg, auth)

		return response, err
	}

	// This comm is not authenticated as servers call this early in their
	// lifecycle to check if they've already registered
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
