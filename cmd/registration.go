////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating callbacks for hooking into comms library

package cmd

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/registration/database"
	"io/ioutil"
)

// Registration Implementation
var registrationImpl RegistrationImpl

// Hardcoded DSA keypair for registration server
var privateKey *signature.DSAPrivateKey

type RegistrationImpl struct {
	Comms *registration.RegistrationComms
}

type Params struct {
	Address       string
	CertPath      string
	KeyPath       string
	NdfOutputPath string
}

type connectionID string

func (c connectionID) String() string {
	return (string)(c)
}

// Configure and start the Permissioning Server
func StartRegistration(params Params) {

	// Read in TLS keys from files
	cert, err := ioutil.ReadFile(params.CertPath)
	if err != nil {
		jww.ERROR.Printf("failed to read certificate at %s: %+v", params.CertPath, err)
	}
	key, err := ioutil.ReadFile(params.KeyPath)
	if err != nil {
		jww.ERROR.Printf("failed to read key at %s: %+v", params.KeyPath, err)
	}

	// Start the communication server
	registrationImpl.Comms = registration.StartRegistrationServer(params.Address,
		NewRegistrationImpl(), cert, key)

	// Wait forever to prevent process from ending
	select {}
}

// Saves the DSA public key to a JSON file
// and returns registation implementation
func NewRegistrationImpl() *RegistrationImpl {
	return &RegistrationImpl{}
}

// Handle registration attempt by a Client
func (m *RegistrationImpl) RegisterUser(registrationCode string, Y, P, Q,
	G []byte) (hash, R, S []byte, err error) {

	// Check database to verify given registration code
	err = database.PermissioningDb.UseCode(registrationCode)
	if err != nil {
		// Invalid registration code, return an error
		jww.ERROR.Printf("Error validating registration code: %s", err)
		return make([]byte, 0), make([]byte, 0), make([]byte, 0), err
	}

	// Concatenate Client public key byte slices
	data := make([]byte, 0)
	data = append(data, Y...)
	data = append(data, P...)
	data = append(data, Q...)
	data = append(data, G...)

	// Use hardcoded keypair to sign Client-provided public key
	sig, err := privateKey.Sign(data, rand.Reader)
	if err != nil {
		// Unable to sign public key, return an error
		jww.ERROR.Printf("Error signing client public key: %s", err)
		return make([]byte, 0), make([]byte, 0), make([]byte, 0),
			errors.New("unable to sign client public key")
	}

	// Return signed public key to Client with empty error field
	jww.INFO.Printf("Verification complete for registration code %s",
		registrationCode)
	return data, sig.R.Bytes(), sig.S.Bytes(), nil
}

// Handle registration attempt by a Node
func (m *RegistrationImpl) RegisterNode(ID []byte, NodeTLSCert,
	GatewayTLSCert, RegistrationCode, Addr string) error {

	// Attempt to insert Node into the database
	err := database.PermissioningDb.InsertNode(ID, RegistrationCode, Addr, NodeTLSCert, GatewayTLSCert)
	if err != nil {
		jww.ERROR.Printf("Unable to insert node: %+v", err)
		return err
	}

	// Obtain the number of registered nodes
	numNodes, err := database.PermissioningDb.CountRegisteredNodes()
	if err != nil {
		jww.ERROR.Printf("Unable to count registered Nodes: %+v", err)
		return err
	}

	// If all nodes have registered
	if numNodes == len(RegistrationCodes) {
		// Finish the node registration process in another thread
		go completeNodeRegistration()
	}
	return nil
}

// Wrapper for completeNodeRegistrationHelper() error-handling
func completeNodeRegistration() {
	err := completeNodeRegistrationHelper()
	if err != nil {
		jww.FATAL.Panicf("Error completing node registration: %+v", err)
	}
}

// Once all nodes have registered, this function is triggered
// to assemble and broadcast the completed topology to all nodes
func completeNodeRegistrationHelper() error {
	// Assemble the completed topology
	var topology []*mixmessages.NodeInfo
	for index, registrationCode := range RegistrationCodes {
		// Get node information for each registration code
		dbNodeInfo, err := database.PermissioningDb.GetNode(registrationCode)
		if err != nil {
			return errors.New(fmt.Sprintf(
				"unable to obtain node for registration"+
					" code %s: %+v", registrationCode, err))
		}
		topology = append(topology, getNodeInfo(dbNodeInfo, index))
	}
	nodeTopology := mixmessages.NodeTopology{
		Topology: topology,
	}

	// Output the completed topology to a JSON file
	err := outputNodeTopologyToJSON(nodeTopology, RegParams.NdfOutputPath)
	if err != nil {
		return errors.New(fmt.Sprintf("unable to output NDF JSON file: %+v",
			err))
	}

	// Broadcast completed topology to all nodes
	jww.INFO.Printf("Broadcasting node topology: %+v", topology)
	for _, nodeInfo := range nodeTopology.Topology {
		err = registrationImpl.Comms.SendNodeTopology(connectionID(nodeInfo.
			Id), &nodeTopology)
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
		Id:        dbNodeInfo.Id,
		Index:     uint32(index),
		IpAddress: dbNodeInfo.Address,
		TlsCert:   dbNodeInfo.NodeCertificate,
	}
	return &nodeInfo
}

// outputNodeTopologyToJSON encodes the NodeTopology structure to JSON and
// outputs it to the specified file path. An error is returned if the JSON
// marshaling fails or if the JSON file cannot be created.
func outputNodeTopologyToJSON(topology mixmessages.NodeTopology, filePath string) error {
	// Generate JSON from structure
	data, err := json.MarshalIndent(topology, "", "\t")
	if err != nil {
		return err
	}

	// Write JSON to file
	err = ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}

	return nil
}
