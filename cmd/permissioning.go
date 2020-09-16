////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating node registration callbacks for hooking into comms library

package cmd

import (
	"bytes"
	gorsa "crypto/rsa"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/xx"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/crypto/tls"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"sort"
	"sync/atomic"
)

// Handle registration check attempt by node. We assume
//  the code being searched for is the node's.
func (m *RegistrationImpl) CheckNodeRegistration(msg *mixmessages.RegisteredNodeCheck) (bool, error) {
	//do edge check to ensure the message is not nil
	if msg == nil {
		return false, errors.Errorf("Message payload for registration check " +
			"is nil. Check could not be processed")
	}

	nodeID, err := id.Unmarshal(msg.ID)
	if err != nil {
		return false, errors.Errorf("Message payload for registration check " +
			"contains invalid ID. Check could not be processed")
	}

	// Check that the node hasn't already been registered. If there is an error,
	//  then the code being checked is either invalid or not registered.
	// There is no need to return database error to node
	nodeInfo, err := storage.PermissioningDb.GetNodeById(nodeID)
	if err != nil {
		return false, nil
	}

	// If the node's ID and Salt are not empty, then the node has been registered
	if !bytes.Equal(nodeInfo.Id, []byte("")) && len(nodeInfo.Salt) > 0 {
		return true, nil
	}

	// Otherwise the code has not been registered
	return false, nil

}

// Handle registration attempt by a Node
func (m *RegistrationImpl) RegisterNode(salt []byte, serverAddr, serverTlsCert, gatewayAddr,
	gatewayTlsCert, registrationCode string) error {

	// Check that the node hasn't already been registered
	nodeInfo, err := storage.PermissioningDb.GetNode(registrationCode)
	if err != nil {
		return errors.Errorf(
			"Registration code %+v is invalid or not currently enabled: %+v", registrationCode, err)
	}

	// Generate the Node ID
	tlsCert, err := tls.LoadCertificate(serverTlsCert)
	if err != nil {
		return errors.Errorf("Could not decode server certificate into a tls cert: %v", err)
	}
	nodePubKey := &rsa.PublicKey{PublicKey: *tlsCert.PublicKey.(*gorsa.PublicKey)}
	if len(salt) > 32 {
		salt = salt[:32]
	}
	nodeId, err := xx.NewID(nodePubKey, salt, id.Node)
	if err != nil {
		return errors.Errorf("Unable to generate Node ID with salt %v: %+v", salt, err)
	}

	// Handle various re-registration cases
	if len(nodeInfo.Id) != 0 {

		// Ensure that generated ID matches stored ID
		// Ensure that salt is not already stored
		if !bytes.Equal(nodeInfo.Id, nodeId.Marshal()) {
			return errors.Errorf("Generated ID %+v does not match stored ID: %+v", nodeId.Marshal(), nodeInfo.Id)

		} else if len(nodeInfo.Salt) != 0 {
			return errors.Errorf(
				"Node with registration code %s has already been registered", registrationCode)
		}

		// Store the newly-provided salt
		return storage.PermissioningDb.UpdateSalt(nodeId, salt)
	}

	// Attempt to insert Node into the database
	err = storage.PermissioningDb.RegisterNode(nodeId, salt, registrationCode, serverAddr,
		serverTlsCert, gatewayAddr, gatewayTlsCert)
	if err != nil {
		return errors.Errorf("unable to insert node: %+v", err)
	}
	jww.DEBUG.Printf("Inserted node %s into the database with code %s",
		nodeId.String(), registrationCode)

	//add the node to the host object for authenticated communications
	_, err = m.Comms.AddHost(nodeId, serverAddr, []byte(serverTlsCert), false, true)
	if err != nil {
		return errors.Errorf("Could not register host for Server %s: %+v", serverAddr, err)
	}

	//add the node to the node map to track its state
	err = m.State.GetNodeMap().AddNode(nodeId, nodeInfo.Sequence, serverAddr, gatewayAddr, nodeInfo.ApplicationId)
	if err != nil {
		return errors.WithMessage(err, "Could not register node with "+
			"state tracker")
	}

	// Notify registration thread
	return m.completeNodeRegistration(registrationCode)
}

// Loads all registered nodes and puts them into the host object and node map.
// Should be run on startup.
func (m *RegistrationImpl) LoadAllRegisteredNodes() error {
	// TODO: This code could probably use some cleanup
	// TODO: We might consider refactoring the ban timer code and this code to share stuff, they might have similar goals.
	nodes, err := storage.PermissioningDb.GetNodesByStatus(node.Active)
	if err != nil {
		return err
	}

	for _, n := range nodes {
		nid, err := id.Unmarshal(n.Id)

		//add the node to the host object for authenticated communications
		_, err = m.Comms.AddHost(nid, n.ServerAddress, []byte(n.NodeCertificate), false, true)
		if err != nil {
			return errors.Errorf("Could not register host for Server %s: %+v", n.ServerAddress, err)
		}

		//add the node to the node map to track its state
		err = m.State.GetNodeMap().AddNode(nid, n.Sequence, n.ServerAddress, n.GatewayAddress, n.ApplicationId)
		if err != nil {
			return errors.WithMessage(err, "Could not register node with "+
				"state tracker")
		}

		err = m.completeNodeRegistration(n.Code)
		if err != nil {
			return err
		}
	}

	bannedNodes, err := storage.PermissioningDb.GetNodesByStatus(node.Banned)
	if err != nil {
		return err
	}

	for _, n := range bannedNodes {
		nid, err := id.Unmarshal(n.Id)

		//add the node to the host object for authenticated communications
		_, err = m.Comms.AddHost(nid, n.ServerAddress, []byte(n.NodeCertificate), false, true)
		if err != nil {
			return errors.Errorf("Could not register host for Server %s: %+v", n.ServerAddress, err)
		}

		//add the node to the node map to track its state
		err = m.State.GetNodeMap().AddBannedNode(nid, n.Sequence, n.ServerAddress, n.GatewayAddress)
		if err != nil {
			return errors.WithMessage(err, "Could not register node with "+
				"state tracker")
		}

		err = m.completeNodeRegistration(n.Code)
		if err != nil {
			return err
		}
	}

	return nil
}

// Handles including new registrations in the network
// fixme: we should split this function into what is relevant to registering a  node and what is relevant
//  to permissioning
func (m *RegistrationImpl) completeNodeRegistration(regCode string) error {

	m.registrationLock.Lock()
	defer m.registrationLock.Unlock()

	m.numRegistered++

	jww.INFO.Printf("Registered %d node(s)!", m.numRegistered)

	// Add the new node to the topology
	m.NDFLock.Lock()
	networkDef := m.State.GetFullNdf().Get()
	gateway, n, regTime, err := assembleNdf(regCode)
	nodeID, err := id.Unmarshal(n.ID)
	if err != nil {
		return errors.Errorf("Error parsing node ID: %v", err)
	}
	m.registrationTimes[nodeID] = regTime

	if err != nil {
		m.NDFLock.Unlock()
		err := errors.Errorf("unable to assemble topology: %+v", err)
		jww.ERROR.Print(err.Error())
		return errors.Errorf("Could not complete registration: %+v", err)
	}

	// update the internal state with the newly-updated ndf
	err = m.State.UpdateNdf(networkDef)
	m.NDFLock.Unlock()
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
		atomic.CompareAndSwapUint32(m.NdfReady, 0, 1)

		jww.INFO.Printf("Minimum number of nodes %d registered for scheduling!", m.numRegistered)

		//signal that scheduling should begin
		m.beginScheduling <- struct{}{}
	}

	return nil
}

func (m *RegistrationImpl) insertNdf(definition *ndf.NetworkDefinition, g ndf.Gateway,
	n ndf.Node, regTime int64) error {
	index := sort.Search(len(definition.Nodes), func(i int) bool {
		nid, _ := id.Unmarshal(definition.Nodes[i].ID)
		return m.registrationTimes[nid] >= regTime
	})
	if index < len(definition.Nodes) && bytes.Compare(definition.Nodes[index].ID, n.ID) == 0 {
		definition.Nodes[index] = n
		definition.Gateways[index] = g
	} else {
		cmpId, err := id.Unmarshal(definition.Nodes[index].ID)
		if err != nil {
			return errors.Errorf("Faield to unmarshal ID from definition: %+v", err)
		}
		nid, err := id.Unmarshal(n.ID)
		if err != nil {
			return errors.Errorf("Failed to unmarshal ID: %+v", err)
		}
		if m.registrationTimes[cmpId] > m.registrationTimes[nid] {
			definition.Nodes = append(definition.Nodes[0:index], append([]ndf.Node{n}, definition.Nodes[index:]...)...)
			definition.Gateways = append(definition.Gateways[0:index], append([]ndf.Gateway{g}, definition.Gateways[index:]...)...)
		} else {
			index++
			definition.Nodes = append(definition.Nodes[0:index], append([]ndf.Node{n}, definition.Nodes[index:]...)...)
			definition.Gateways = append(definition.Gateways[0:index], append([]ndf.Gateway{g}, definition.Gateways[index:]...)...)
		}
	}
	return nil
}

// helper function which appends the ndf to the maximum order
func appendNdf(definition *ndf.NetworkDefinition, order int) {
	// Avoid causing a divide by zero panic if both order and definition.Nodes is zero, 0 % 0 is incalculable
	lengthDifference := 0
	if order == 0 && len(definition.Nodes) == 0 {
		lengthDifference = 1
	} else {
		lengthDifference = (order - len(definition.Nodes)) + 1
	}

	gwExtension := make([]ndf.Gateway, lengthDifference)
	nodeExtension := make([]ndf.Node, lengthDifference)
	definition.Nodes = append(definition.Nodes, nodeExtension...)
	definition.Gateways = append(definition.Gateways, gwExtension...)

}

// Assemble information for the given registration code
func assembleNdf(code string) (ndf.Gateway, ndf.Node, int64, error) {

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

	n := ndf.Node{
		ID:             nodeID.Bytes(),
		Address:        nodeInfo.ServerAddress,
		TlsCertificate: nodeInfo.NodeCertificate,
	}

	jww.INFO.Printf("Node %s (AppID: %d) registered with code %s", nodeID, nodeInfo.ApplicationId, code)

	gwID := nodeID.DeepCopy()
	gwID.SetType(id.Gateway)
	gateway := ndf.Gateway{
		ID:             gwID.Bytes(),
		Address:        nodeInfo.GatewayAddress,
		TlsCertificate: nodeInfo.GatewayCertificate,
	}

	return gateway, n, nodeInfo.DateRegistered.Unix(), nil
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
