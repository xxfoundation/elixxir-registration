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
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/certAuthority"
	"gitlab.com/elixxir/registration/storage"
	"sync/atomic"
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
	m.nodeCompleted <- RegistrationCode
	return nil
}

// Handles including new registrations in the network
func (m *RegistrationImpl) nodeRegistrationCompleter() error {

	// Wait for Nodes to complete registration
	for numNodes := 0; numNodes < len(RegistrationCodes); numNodes++ {
		jww.INFO.Printf("Registered %d node(s)!", numNodes)
		regCode := <-m.nodeCompleted

		// Add the new node to the topology
		networkDef := m.State.GetFullNdf().Get()
		gateway, node, err := assembleNdf(regCode)
		if err != nil {
			return errors.Errorf("unable to assemble topology: %+v", err)
		}
		networkDef.Gateways = append(networkDef.Gateways, *gateway)
		networkDef.Nodes = append(networkDef.Nodes, *node)

		// Update the internal state with the newly-updated ndf
		err = m.State.UpdateNdf(networkDef)
		if err != nil {
			return err
		}

		// Output the current topology to a JSON file
		err = outputToJSON(networkDef, m.ndfOutputPath)
		if err != nil {
			return errors.Errorf("unable to output NDF JSON file: %+v", err)
		}

		// Kick off the network if the minimum number of nodes has been met
		if uint32(numNodes) == m.params.minimumNodes-1 {
			jww.INFO.Printf("Minimum number of nodes registered!")

			// Create the first round
			err = m.createNextRound()
			if err != nil {
				return err
			}

			// Alert that the network is ready
			atomic.CompareAndSwapUint32(m.NdfReady, 0, 1)
		}
	}
	return nil
}

// Assemble information for the given registration code
func assembleNdf(code string) (*ndf.Gateway, *ndf.Node, error) {

	// Get node information for each registration code
	nodeInfo, err := storage.PermissioningDb.GetNode(code)
	if err != nil {
		return nil, nil, errors.Errorf(
			"unable to obtain node for registration"+
				" code %+v: %+v", code, err)
	}
	var node ndf.Node
	node.ID = nodeInfo.Id
	node.TlsCertificate = nodeInfo.NodeCertificate
	node.Address = nodeInfo.ServerAddress

	var gateway ndf.Gateway
	gateway.TlsCertificate = nodeInfo.GatewayCertificate
	gateway.Address = nodeInfo.GatewayAddress

	jww.DEBUG.Printf("Assembled the network topology")
	return &gateway, &node, nil
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
