////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating node registration callbacks for hooking into comms library

package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/certAuthority"
	"gitlab.com/elixxir/registration/database"
)

// Handle registration attempt by a Node
func (m *RegistrationImpl) RegisterNode(ID []byte, ServerAddr, ServerTlsCert,
	GatewayAddr, GatewayTlsCert, RegistrationCode string) error {

	// Check that the node hasn't already been registered
	nodeInfo, err := database.PermissioningDb.GetNode(RegistrationCode)
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"Registration code %+v is invalid or not currently enabled: %+v", RegistrationCode, err))
		jww.ERROR.Printf("%+v", errMsg)
		return errMsg
	}
	if nodeInfo.Id != nil {
		errMsg := errors.New(fmt.Sprintf(
			"Node with registration code %+v has already been registered", RegistrationCode))
		jww.ERROR.Printf("%+v", errMsg)
		return errMsg
	}

	// Connect back to the Node using the provided certificate
	err = m.Comms.ConnectToRemote(id.NewNodeFromBytes(ID), ServerAddr,
		[]byte(ServerTlsCert), false)
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"Failed to return connection to Node: %+v", err))
		jww.ERROR.Printf("%+v", errMsg)
		return errMsg
	}
	jww.INFO.Printf("Connected to node %+v at address %+v", ID, ServerAddr)

	// Load the node and gateway certs
	nodeCertificate, err := tls.LoadCertificate(ServerTlsCert)
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"Failed to load node certificate: %v", err))
		jww.ERROR.Printf("%v", errMsg)
		return errMsg
	}
	gatewayCertificate, err := tls.LoadCertificate(GatewayTlsCert)
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"Failed to load gateway certificate: %v", err))
		jww.ERROR.Printf("%v", errMsg)
		return errMsg
	}

	// Sign the node and gateway certs
	signedNodeCert, err := certAuthority.Sign(nodeCertificate, m.permissioningCert, &(m.permissioningKey.PrivateKey))
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"failed to sign node certificate: %v", err))
		jww.ERROR.Printf("%v", errMsg)
		return errMsg
	}
	signedGatewayCert, err := certAuthority.Sign(gatewayCertificate, m.permissioningCert, &(m.permissioningKey.PrivateKey))
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"Failed to sign gateway certificate: %v", err))
		jww.ERROR.Printf("%v", errMsg)
		return errMsg
	}

	// Attempt to insert Node into the database
	err = database.PermissioningDb.InsertNode(ID, RegistrationCode, ServerAddr,
		signedNodeCert, GatewayAddr, signedGatewayCert)
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"unable to insert node: %+v", err))
		jww.ERROR.Printf("%+v", errMsg)
		return errMsg
	}
	jww.DEBUG.Printf("Inserted node %+v into the database with code %+v",
		ID, RegistrationCode)

	// Obtain the number of registered nodes
	_, err = database.PermissioningDb.CountRegisteredNodes()
	if err != nil {
		errMsg := errors.New(fmt.Sprintf(
			"Unable to count registered Nodes: %+v", err))
		jww.ERROR.Printf("%+v", errMsg)
		return errMsg
	}

	m.completedNodes <- struct{}{}
	return nil
}

// Wrapper for completed node registration error handling
func nodeRegistrationCompleter(impl *RegistrationImpl) {
	// Wait for all Nodes to complete registration
	for numNodes := 0; numNodes < impl.NumNodesInNet; numNodes++ {
		<-impl.completedNodes
	}
	//Assemble ndf here as well??
	// Assemble the completed topology
	topology, gateways, nodes, err := assembleTopology(RegistrationCodes)
	if err != nil {
		jww.FATAL.Printf("unable to assemble topology: %+v", err)
	}

	registration, err := assembleRegistration()
	if err != nil {
		jww.FATAL.Printf("Failed to build registration for ndf: %v", err)
	}

	//Assemble a different ndf
	networkDef := &ndf.NetworkDefinition{
		Registration: registration,
		Nodes:        nodes,
		Gateways:     gateways,
		UDB:          udbParams,
	}
	fmt.Printf("node reg complete: \n %v", gateways)
	impl.ndfData = networkDef

	err = outputToJSON(impl.ndfData, impl.ndfOutputPath)
	if err != nil {
		errMsg := errors.New(fmt.Sprintf("unable to output NDF JSON file: %+v",
			err))
		jww.FATAL.Printf(errMsg.Error())
	}

	hash := sha256.New()

	ndfBytes := serializeNdf(networkDef)
	hash.Write(ndfBytes)
	impl.ndfHash = hash.Sum(nil)

	// Broadcast completed topology to all nodes
	err = broadcastTopology(impl, topology)
	if err != nil {
		jww.FATAL.Panicf("Error completing node registration: %+v", err)
	}

	jww.INFO.Printf("Node registration complete!")
}

//Assemble the registration information
func assembleRegistration() (ndf.Registration, error) {
	var reg ndf.Registration
	reg.Address = RegParams.Address
	certBytes, err := utils.ReadFile(RegParams.CertPath)
	if err != nil {
		return ndf.Registration{}, errors.New(fmt.Sprintf("unable to read certificate path: %v", err))
	}
	reg.TlsCertificate = string(certBytes)

	return reg, nil
}

// Assemble the completed topology from the database
func assembleTopology(codes []string) (*mixmessages.NodeTopology, []ndf.Gateway, []ndf.Node, error) {
	var topology []*mixmessages.NodeInfo
	var gateways []ndf.Gateway
	var nodes []ndf.Node
	for index, registrationCode := range codes {
		// Get node information for each registration code
		dbNodeInfo, err := database.PermissioningDb.GetNode(registrationCode)
		if err != nil {
			return nil, nil, nil, errors.New(fmt.Sprintf(
				"unable to obtain node for registration"+
					" code %+v: %+v", registrationCode, err))
		}
		var node ndf.Node
		node.ID = dbNodeInfo.Id

		var gateway ndf.Gateway
		gateway.TlsCertificate = dbNodeInfo.GatewayCertificate
		gateway.Address = dbNodeInfo.GatewayAddress

		topology = append(topology, getNodeInfo(dbNodeInfo, index))
		gateways = append(gateways, gateway)
		nodes = append(nodes, node)
	}
	nodeTopology := mixmessages.NodeTopology{
		Topology: topology,
	}
	jww.DEBUG.Printf("Assembled the topology")
	return &nodeTopology, gateways, nodes, nil
}

// Broadcast completed topology to all nodes
func broadcastTopology(impl *RegistrationImpl, topology *mixmessages.NodeTopology) error {
	jww.INFO.Printf("Broadcasting node topology: %+v", topology)
	for _, nodeInfo := range topology.Topology {
		err := impl.Comms.SendNodeTopology(id.NewNodeFromBytes(nodeInfo.Id), topology)
		if err != nil {
			return errors.New(fmt.Sprintf(
				"unable to broadcast node topology: %+v", err))
		}
	}
	return nil
}

// getNodeInfo creates a NodeInfo message from the
// node info in the database and other input params
func getNodeInfo(dbNodeInfo *database.NodeInformation, index int) *mixmessages.NodeInfo {
	nodeInfo := mixmessages.NodeInfo{
		Id:             dbNodeInfo.Id,
		Index:          uint32(index),
		ServerAddress:  dbNodeInfo.ServerAddress,
		ServerTlsCert:  dbNodeInfo.NodeCertificate,
		GatewayAddress: dbNodeInfo.GatewayAddress,
		GatewayTlsCert: dbNodeInfo.GatewayCertificate,
	}
	return &nodeInfo
}

// outputNodeTopologyToJSON encodes the NodeTopology structure to JSON and
// outputs it to the specified file path. An error is returned if the JSON
// marshaling fails or if the JSON file cannot be created.
func outputToJSON(ndfData *ndf.NetworkDefinition, filePath string) error {
	// Generate JSON from structure
	data, err := json.MarshalIndent(ndfData, "", "\t")
	if err != nil {
		return err
	}
	// Write JSON to file
	err = utils.WriteFile(filePath, data, utils.FilePerms, utils.DirPerms)
	if err != nil {
		return err
	}

	return nil
}
