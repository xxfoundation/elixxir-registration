////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating node registration callbacks for hooking into comms library

package cmd

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/certAuthority"
	"gitlab.com/elixxir/registration/storage"
	"strconv"
	"sync/atomic"
)

// Handle registration attempt by a Node
func (m *RegistrationImpl) RegisterNode(ID *id.ID, ServerAddr, ServerTlsCert,
	GatewayAddr, GatewayTlsCert, RegistrationCode string) error {

	// Check that the node hasn't already been registered
	nodeInfo, err := storage.PermissioningDb.GetNode(RegistrationCode)
	if err != nil {
		return errors.Errorf(
			"Registration code %+v is invalid or not currently enabled: %+v", RegistrationCode, err)
	}
	if !bytes.Equal(nodeInfo.Id, []byte("")) {
		return errors.Errorf(
			"Node with registration code %+v has already been registered", RegistrationCode)
	}
	pk := &m.State.GetPrivateKey().PrivateKey
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
		m.permissioningCert, pk)
	if err != nil {
		return errors.Errorf("failed to sign node certificate: %v", err)
	}
	signedGatewayCert, err := certAuthority.Sign(gatewayCertificate,
		m.permissioningCert, pk)
	if err != nil {
		return errors.Errorf("Failed to sign gateway certificate: %v", err)
	}

	// Attempt to insert Node into the database
	err = storage.PermissioningDb.RegisterNode(ID, RegistrationCode, ServerAddr,
		signedNodeCert, GatewayAddr, signedGatewayCert)
	if err != nil {
		return errors.Errorf("unable to insert node: %+v", err)
	}
	jww.DEBUG.Printf("Inserted node %s into the database with code %s",
		ID.String(), RegistrationCode)

	//add the node to the host object for authenticated communications
	_, err = m.Comms.AddHost(ID, ServerAddr, []byte(ServerTlsCert), false, true)
	if err != nil {
		return errors.Errorf("Could not register host for Server %s: %+v", ServerAddr, err)
	}

	//add the node to the node map to track its state
	err = m.State.GetNodeMap().AddNode(ID, nodeInfo.Sequence, ServerAddr, GatewayAddr)
	if err != nil {
		return errors.WithMessage(err, "Could not register node with "+
			"state tracker")
	}

	// Notify registration thread
	return m.completeNodeRegistration(RegistrationCode)
}

// Handles including new registrations in the network
// fixme: we should split this function into what is relevant to registering a  node and what is relevant
//  to permissioning
func (m *RegistrationImpl) completeNodeRegistration(regCode string) error {

	m.registrationLock.Lock()
	defer m.registrationLock.Unlock()

	m.numRegistered++

	jww.INFO.Printf("Registered %d node(s)! %s", m.numRegistered, regCode)

	// Add the new node to the topology
	networkDef := m.State.GetFullNdf().Get()
	gateway, node, order, err := assembleNdf(regCode)
	if err != nil {
		err := errors.Errorf("unable to assemble topology: %+v", err)
		jww.ERROR.Print(err.Error())
		return errors.Errorf("Could not complete registration: %+v", err)
	}

	if order != -1 {
		// fixme: consider removing. this allows clients to remain agnostic of teaming order
		//  by forcing team order == ndf order for simple non-random
		if order >= len(networkDef.Nodes) {
			appendNdf(networkDef, order)
		}

		networkDef.Gateways[order] = gateway
		networkDef.Nodes[order] = node
	} else {
		networkDef.Gateways = append(networkDef.Gateways, gateway)
		networkDef.Nodes = append(networkDef.Nodes, node)
	}

	// update the internal state with the newly-updated ndf
	err = m.State.UpdateNdf(networkDef)
	if err != nil {
		return err
	}

	// Output the current topology to a JSON file
	err = outputToJSON(networkDef, m.ndfOutputPath)
	if err != nil {
		err := errors.Errorf("unable to output NDF JSON file: %+v", err)
		jww.ERROR.Print(err.Error())
		return errors.Errorf("Could not complete registration: %+v", err)
	}

	// Kick off the network if the minimum number of nodes has been met
	if uint32(m.numRegistered) == m.params.minimumNodes {
		jww.INFO.Printf("Minimum number of nodes %d registered!", m.numRegistered)

		atomic.CompareAndSwapUint32(m.NdfReady, 0, 1)

		//signal that scheduling should begin
		m.beginScheduling <- struct{}{}
	}

	return nil
}

// helper function which appends the ndf to the maximum order
func appendNdf(definition *ndf.NetworkDefinition, order int) {
	lengthDifference := (order % len(definition.Nodes)) + 1
	gwExtension := make([]ndf.Gateway, lengthDifference)
	nodeExtension := make([]ndf.Node, lengthDifference)
	definition.Nodes = append(definition.Nodes, nodeExtension...)
	definition.Gateways = append(definition.Gateways, gwExtension...)

}

// Assemble information for the given registration code
func assembleNdf(code string) (ndf.Gateway, ndf.Node, int, error) {

	// Get node information for each registration code
	nodeInfo, err := storage.PermissioningDb.GetNode(code)
	if err != nil {
		return ndf.Gateway{}, ndf.Node{}, 0, errors.Errorf(
			"unable to obtain node for registration"+
				" code %+v: %+v", code, err)
	}

	nodeID, err := id.Unmarshal(nodeInfo.Id)
	if err != nil {
		return ndf.Gateway{}, ndf.Node{}, 0, errors.Errorf("Error parsing node ID: %v", err)
	}

	node := ndf.Node{
		ID:             nodeID.Bytes(),
		Address:        nodeInfo.ServerAddress,
		TlsCertificate: nodeInfo.NodeCertificate,
	}

	gwID := nodeID.DeepCopy()
	gwID.SetType(id.Gateway)
	gateway := ndf.Gateway{
		ID:             gwID.Bytes(),
		Address:        nodeInfo.GatewayAddress,
		TlsCertificate: nodeInfo.GatewayCertificate,
	}

	order, err := strconv.Atoi(nodeInfo.Sequence)
	if err != nil {
		return gateway, node, -1, nil
	}

	return gateway, node, order, nil
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
