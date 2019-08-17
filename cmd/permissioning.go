////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating node registration callbacks for hooking into comms library

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/utils"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/certAuthority"
	"gitlab.com/elixxir/registration/database"
	"io/ioutil"
)

// Handle registration attempt by a Node
func (m *RegistrationImpl) RegisterNode(ID []byte, ServerTlsCert,
	GatewayTlsCert, RegistrationCode, Addr string) error {
	// Connect back to the Node using the provided certificate
	err := m.Comms.ConnectToRemote(id.NewNodeFromBytes(ID), Addr,
		[]byte(ServerTlsCert))
	if err != nil {
		jww.ERROR.Printf("Failed to return connection to Node: %+v", err)
		return err
	}
	// Load the node and gateway certs
	nodeCertificate, err := tls.LoadCertificate(ServerTlsCert)
	if err != nil {
		jww.ERROR.Printf("Failed to load node certificate: %v", err)
		return err
	}
	gatewayCertificate, err := tls.LoadCertificate(GatewayTlsCert)
	if err != nil {
		jww.ERROR.Printf("Failed to load gateway certificate: %v", err)
		return err
	}

	// Sign the node and gateway certs
	signedNodeCert, err := certAuthority.Sign(nodeCertificate, m.permissioningCert, &(m.permissioningKey.PrivateKey))
	if err != nil {
		jww.ERROR.Printf("failed to sign node certificate: %v", err)
		return err
	}
	//Sign the gateway cert reqs
	signedGatewayCert, err := certAuthority.Sign(gatewayCertificate, m.permissioningCert, &(m.permissioningKey.PrivateKey))
	if err != nil {
		jww.ERROR.Printf("Failed to sign gateway certificate: %v", err)
		return err
	}
	// Attempt to insert Node into the database
	err = database.PermissioningDb.InsertNode(ID, RegistrationCode, Addr, signedNodeCert, signedGatewayCert)
	if err != nil {
		jww.ERROR.Printf("unable to insert node: %+v", err)
		return err
	}
	// Obtain the number of registered nodes
	numNodes, err := database.PermissioningDb.CountRegisteredNodes()
	if err != nil {
		jww.ERROR.Printf("Unable to count registered Nodes: %+v", err)
		return err
	}

	runFunc := func() {
		go NodeRegistrationCompleter(m)
		m.completedNodes <- m.Comms

	}

	// If all nodes have registered
	if numNodes == len(RegistrationCodes) {

		// Finish the node registration process in another thread
		go runFunc()
	}
	return nil
}

// Wrapper for completeNodeRegistrationHelper() error-handling
func NodeRegistrationCompleter(impl *RegistrationImpl) {
	//var tmp *registration.RegistrationComms
	//doubt that you need something this complex.. you might only need a channel of one
	//you only need one if you are going to use one impl to make all the connections
	//if you are having an impl for every conn, you are gonna have to pass the map for each..add to some data struct
	//But I doubt that this is the case (the multi impl's)
	//wtf
	//
	impl.Comms = <-impl.completedNodes
	err := completeNodeRegistrationHelper(impl)
	if err != nil {
		jww.FATAL.Panicf("Error completing node registration: %+v", err)
	}
	jww.INFO.Printf("Node registration complete!")
}

// Once all nodes have registered, this function is triggered
// to assemble and broadcast the completed topology to all nodes
func completeNodeRegistrationHelper(impl *RegistrationImpl) error {
	// Assemble the completed topology
	topology, err := assembleTopology(RegistrationCodes)
	if err != nil {
		return err
	}

	// Output the completed topology to a JSON file
	err = outputNodeTopologyToJSON(topology, RegParams.NdfOutputPath)
	if err != nil {
		return errors.New(fmt.Sprintf("unable to output NDF JSON file: %+v",
			err))
	}
	// Broadcast completed topology to all nodes
	return broadcastTopology(impl, topology)
}

// Assemble the completed topology from the database
func assembleTopology(codes []string) (*mixmessages.NodeTopology, error) {
	var topology []*mixmessages.NodeInfo
	for index, registrationCode := range codes {
		// Get node information for each registration code
		dbNodeInfo, err := database.PermissioningDb.GetNode(registrationCode)
		if err != nil {
			return nil, errors.New(fmt.Sprintf(
				"unable to obtain node for registration"+
					" code %s: %+v", registrationCode, err))
		}
		topology = append(topology, getNodeInfo(dbNodeInfo, index))
	}
	nodeTopology := mixmessages.NodeTopology{
		Topology: topology,
	}
	return &nodeTopology, nil
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

// getNodeInfo creates a NodeInfo mixmessage from the
// node info in the database and other input params
func getNodeInfo(dbNodeInfo *database.NodeInformation, index int) *mixmessages.NodeInfo {
	nodeInfo := mixmessages.NodeInfo{
		Id:             dbNodeInfo.Id,
		Index:          uint32(index),
		IpAddress:      dbNodeInfo.Address,
		ServerTlsCert:  dbNodeInfo.NodeCertificate,
		GatewayTlsCert: dbNodeInfo.GatewayCertificate,
	}
	return &nodeInfo
}

// outputNodeTopologyToJSON encodes the NodeTopology structure to JSON and
// outputs it to the specified file path. An error is returned if the JSON
// marshaling fails or if the JSON file cannot be created.
func outputNodeTopologyToJSON(topology *mixmessages.NodeTopology, filePath string) error {
	// Generate JSON from structure
	data, err := json.MarshalIndent(topology, "", "\t")
	if err != nil {
		return err
	}

	// Write JSON to file
	err = ioutil.WriteFile(utils.GetFullPath(filePath), data, 0644)
	if err != nil {
		return err
	}

	return nil
}
