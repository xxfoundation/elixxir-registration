////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating node registration callbacks for hooking into comms library

package cmd

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/certAuthority"
	"gitlab.com/elixxir/registration/storage"
	"sync/atomic"
	"time"
)

// Handle registration attempt by a Node
func (m *RegistrationImpl) RegisterNode(ID []byte, ServerAddr, ServerTlsCert,
	GatewayAddr, GatewayTlsCert, RegistrationCode string) error {

	// Get proper ID string
	idString := id.NewNodeFromBytes(ID).String()

	// Check that the node hasn't already been registered
	nodeInfo, err := storage.PermissioningDb.GetNode(RegistrationCode)
	if err != nil {
		return errors.Errorf(
			"Registration code %+v is invalid or not currently enabled: %+v", RegistrationCode, err)
	}
	if nodeInfo.Id != nil {
		return errors.Errorf(
			"Node with registration code %+v has already been registered", RegistrationCode)
	}

	// Load the node and gateway certs
	nodeCertificate, err := tls.LoadCertificate(ServerTlsCert)
	if err != nil {
		return errors.Errorf("Failed to load node certificate: %v", err)
	}
	gatewayCertificate, err := tls.LoadCertificate(GatewayTlsCert)
	if err != nil {
		return errors.Errorf("Failed to load gateway certificate: %v", err)
	}

	// Sign the node and gateway certs
	signedNodeCert, err := certAuthority.Sign(nodeCertificate,
		m.permissioningCert, &(m.State.PrivateKey.PrivateKey))
	if err != nil {
		return errors.Errorf("failed to sign node certificate: %v", err)
	}
	signedGatewayCert, err := certAuthority.Sign(gatewayCertificate,
		m.permissioningCert, &(m.State.PrivateKey.PrivateKey))
	if err != nil {
		return errors.Errorf("Failed to sign gateway certificate: %v", err)
	}

	// Attempt to insert Node into the database
	err = storage.PermissioningDb.InsertNode(ID, RegistrationCode,
		signedNodeCert, ServerAddr, GatewayAddr, signedGatewayCert)
	if err != nil {
		return errors.Errorf("unable to insert node: %+v", err)
	}
	jww.DEBUG.Printf("Inserted node %s into the database with code %s",
		idString, RegistrationCode)

	// Obtain the number of registered nodes
	_, err = storage.PermissioningDb.CountRegisteredNodes()
	if err != nil {
		return errors.Errorf("Unable to count registered Nodes: %+v", err)
	}

	_, err = m.Comms.AddHost(idString, ServerAddr, []byte(ServerTlsCert), false, true)
	if err != nil {
		return errors.Errorf("Could not register host for Server %s: %+v", ServerAddr, err)
	}

	jww.DEBUG.Printf("Total number of expected nodes for registration"+
		" completion: %d", len(RegistrationCodes))
	m.nodeCompleted <- struct{}{}
	return nil
}

// nodeRegistrationCompleter is a wrapper for completed node registration error handling
func nodeRegistrationCompleter(impl *RegistrationImpl) error {
	// Wait for all Nodes to complete registration
	for numNodes := uint32(0); numNodes < uint32(len(RegistrationCodes)); numNodes++ {
		jww.INFO.Printf("Registered %d node(s)!", numNodes)
		<-impl.nodeCompleted
	}
	// Assemble the completed topology
	gateways, nodes, err := assembleNdf(RegistrationCodes)
	if err != nil {
		return errors.Errorf("unable to assemble topology: %+v", err)
	}

	// Assemble the registration server information
	registration := ndf.Registration{Address: RegParams.publicAddress, TlsCertificate: impl.certFromFile}

	// Construct the NDF
	networkDef := &ndf.NetworkDefinition{
		Registration: registration,
		Timestamp:    time.Now(),
		Nodes:        nodes,
		Gateways:     gateways,
		UDB:          udbParams,
		E2E:          RegParams.e2e,
		CMIX:         RegParams.cmix,
	}

	// Assemble notification server information if configured
	if RegParams.NsCertPath != "" && RegParams.NsAddress != "" {
		nsCert, err := utils.ReadFile(RegParams.NsCertPath)
		if err != nil {
			return errors.Errorf("unable to read notification certificate")
		}
		networkDef.Notification = ndf.Notification{Address: RegParams.NsAddress, TlsCertificate: string(nsCert)}
	} else {
		jww.WARN.Printf("Configured to run without notifications bot!")
	}

	// Update the internal state with the newly-formed NDF
	err = impl.State.UpdateNdf(networkDef)
	err = impl.CreateNextRound()
	if err != nil {
		return err
	}

	// Output the completed topology to a JSON file and save marshall'ed json data
	err = outputToJSON(networkDef, impl.ndfOutputPath)
	if err != nil {
		return errors.Errorf("unable to output NDF JSON file: %+v", err)
	}

	// Alert that registration is complete
	atomic.CompareAndSwapUint32(impl.NdfReady, 0, 1)
	return nil
}

// Assemble the completed topology from the database
func assembleNdf(codes []string) ([]ndf.Gateway, []ndf.Node, error) {
	var gateways []ndf.Gateway
	var nodes []ndf.Node
	for _, registrationCode := range codes {
		// Get node information for each registration code
		nodeInfo, err := storage.PermissioningDb.GetNode(registrationCode)
		if err != nil {
			return nil, nil, errors.Errorf(
				"unable to obtain node for registration"+
					" code %+v: %+v", registrationCode, err)
		}
		var node ndf.Node
		node.ID = nodeInfo.Id
		node.TlsCertificate = nodeInfo.NodeCertificate
		node.Address = nodeInfo.ServerAddress

		var gateway ndf.Gateway
		gateway.TlsCertificate = nodeInfo.GatewayCertificate
		gateway.Address = nodeInfo.GatewayAddress
		//Since we are appending them simultaneously, indexing corresponding
		// gateway-node is just finding your index (as a gateway or a node)
		gateways = append(gateways, gateway)
		nodes = append(nodes, node)
	}
	jww.DEBUG.Printf("Assembled the network topology")
	return gateways, nodes, nil
}

// outputNodeTopologyToJSON encodes the NodeTopology structure to JSON and
// outputs it to the specified file path. An error is returned if the JSON
// marshaling fails or if the JSON file cannot be created.
func outputToJSON(ndfData *ndf.NetworkDefinition, filePath string) error {
	// Generate JSON from structure
	data, err := ndfData.Marshal()
	if err != nil {
		return err
	}
	// Write JSON to file
	return utils.WriteFile(filePath, data, utils.FilePerms, utils.DirPerms)
}

// Attempt to update the internal state after a node polling operation
func (m *RegistrationImpl) UpdateState(id *id.Node, activity *current.Activity) error {
	// Convert node activity to round state
	roundState, err := activity.ConvertToRoundState()
	if err != nil {
		return err
	}

	// Update node state
	m.State.UpdateNodeState(id, roundState)
	return nil
}

// Initiate the next round with a selection of nodes
func (m *RegistrationImpl) CreateNextRound() error {
	// Build a topology (currently consisting of all nodes in network)
	var topology []string
	for _, node := range m.State.GetPartialNdf().Get().Nodes {
		topology = append(topology, node.GetNodeId().String())
	}

	// Progress to the next round
	return m.createNextRound(topology, m.params.batchSize)
}

// Increments the state of the current round if needed
func (m *RegistrationImpl) incrementRoundState(state states.Round) error {
	s := m.State

	// Handle state transitions
	switch state {
	case states.STANDBY:
		s.CurrentRound.State = uint32(states.REALTIME)
		// Handle timestamp edge case with realtime
		s.CurrentRound.Timestamps[states.REALTIME] = uint64(time.Now().Add(2 * time.Second).Unix())
	case states.COMPLETED:
		s.CurrentRound.State = uint32(states.COMPLETED)
		s.CurrentRound.Timestamps[states.COMPLETED] = uint64(time.Now().Unix())
	default:
		return nil
	}
	// Update current round state
	s.CurrentUpdate += 1
	s.CurrentRound.UpdateID = uint64(s.CurrentUpdate)

	// Sign the new round object
	err := signature.Sign(s.CurrentRound.RoundInfo, s.PrivateKey)
	if err != nil {
		return err
	}

	// Insert update into the state tracker
	return s.AddRoundUpdate(s.CurrentRound.RoundInfo)
}

// Builds and inserts the next RoundInfo object into the internal state
func (m *RegistrationImpl) createNextRound(topology []string, batchSize uint32) error {
	s := m.State

	// Build the new current round object
	s.CurrentUpdate += 1
	s.CurrentRound.RoundInfo = &pb.RoundInfo{
		ID:         uint64(s.RoundData.GetLastRoundID() + 1),
		UpdateID:   uint64(s.CurrentUpdate),
		State:      uint32(states.PRECOMPUTING),
		BatchSize:  batchSize,
		Topology:   topology,
		Timestamps: make([]uint64, states.NUM_STATES),
	}
	s.CurrentRound.Timestamps[states.PRECOMPUTING] = uint64(time.Now().Unix())
	jww.DEBUG.Printf("Initializing round %d...", s.CurrentRound.ID)

	// Initialize network status
	for i := range s.CurrentRound.NetworkStatus {
		val := uint32(0)
		s.CurrentRound.NetworkStatus[i] = &val
	}

	// Initialize node states based on given topology
	for _, nodeId := range topology {
		newState := uint32(states.PENDING)
		s.CurrentRound.NodeStatuses[*id.NewNodeFromBytes([]byte(
			nodeId))] = &newState
	}

	// Sign the new round object
	err := signature.Sign(s.CurrentRound.RoundInfo, s.PrivateKey)
	if err != nil {
		return err
	}

	// Insert the new round object into the state tracker
	err = s.RoundData.UpsertRound(s.CurrentRound.RoundInfo)
	if err != nil {
		return err
	}
	return s.AddRoundUpdate(s.CurrentRound.RoundInfo)
}

// Control thread for advancement of network state
func (m *RegistrationImpl) StateControl() {
	s := m.State
	for range s.Update {

		// Handle completion of a round
		if s.GetCurrentRoundState() == states.COMPLETED {
			// Create the new round
			err := m.createNextRound(s.CurrentRound.GetTopology(), m.params.batchSize)
			if err != nil {
				// TODO: Error handling
				jww.FATAL.Panicf("Unable to create next round: %+v", err)
			}
		}

		// Check whether the round state is ready to increment
		nextState := states.Round(s.CurrentRound.State + 1)
		numNodesInRound := uint32(len(s.CurrentRound.Topology))
		if *s.CurrentRound.NetworkStatus[nextState] == numNodesInRound {
			// Increment the round state
			err := m.incrementRoundState(nextState)
			if err != nil {
				// TODO: Error handling
				jww.FATAL.Panicf("Unable to create next round: %+v", err)
			}
		}
	}
}
