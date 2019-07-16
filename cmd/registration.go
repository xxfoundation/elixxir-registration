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
	"github.com/mitchellh/go-homedir"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/registration"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/registration/database"
	"io/ioutil"
)

// Registration Implementation
var registrationImpl RegistrationImpl

// DSA Params
var dsaParams = signature.GetDefaultDSAParams()

// Hardcoded DSA keypair for registration server
var privateKey = signature.ReconstructPrivateKey(
	signature.ReconstructPublicKey(dsaParams,
		large.NewIntFromString("1ae3fd4e9829fb464459e05bca392bec5067152fb43a569ad3c3b68bbcad84f0"+
			"ff8d31c767da3eabcfc0870d82b39568610b52f2b72b493bbede6e952c9a7fd4"+
			"4a8161e62a9046828c4a65f401b2f054ebf7376e89dab547d8a3c3d46891e78a"+
			"cfc4015713cbfb5b0b6cab0f8dfb46b891f3542046ace4cab984d5dfef4f52d4"+
			"347dc7e52f6a7ea851dda076f0ed1fef86ec6b5c2a4807149906bf8e0bf70b30"+
			"1147fea88fd95009edfbe0de8ffc1a864e4b3a24265b61a1c47a4e9307e7c84f"+
			"9b5591765b530f5859fa97b22ce9b51385d3d13088795b2f9fd0cb59357fe938"+
			"346117df2acf2bab22d942de1a70e8d5d62fc0e99d8742a0f16df94ce3a0abbb", 16)),
	large.NewIntFromString("dab0febfab103729077ad4927754f6390e366fdf4c58e8d40dadb3e94c444b54", 16))

type RegistrationImpl struct {
	Comms *registration.RegistrationComms
}

type Params struct {
	Address  string
	CertPath string
	KeyPath  string
}

type connectionID string

func (c connectionID) String() string {
	return (string)(c)
}

// StartRegistration sets up registration server
// and comms and waits forever
func StartRegistration(params Params) {
	registrationImpl := NewRegistrationImpl()

	registrationImpl.Comms = registration.StartRegistrationServer(params.Address, registration.Handler(registrationImpl),
		params.CertPath, params.KeyPath)

	select {}
}

// Saves the DSA public key to a JSON file
// and returns registation implementation
func NewRegistrationImpl() *RegistrationImpl {

	// Get the default parameters and generate a public key from it
	dsaParams := signature.GetDefaultDSAParams()
	publicKey := dsaParams.PrivateKeyGen(rand.Reader).PublicKeyGen()

	// Output the DSA public key to JSON file
	outputDsaPubKeyToJson(publicKey, ".elixxir", "registration_info.json")

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

		// Create node topology
		var topology []*mixmessages.NodeInfo
		for index, registrationCode := range RegistrationCodes {

			dbNodeInfo, err := database.PermissioningDb.GetNode(registrationCode)

			if err != nil {
				return err
			}

			nodeInfo := getNodeInfo(dbNodeInfo, uint32(index), NodeTLSCert)

			topology = append(topology, nodeInfo)
		}

		nodeTopology := mixmessages.NodeTopology{
			Topology: topology,
		}

		// Broadcast to all nodes
		jww.INFO.Printf("INFO: Broadcasting node topology: %+v", topology)
		for _, nodeInfo := range nodeTopology.Topology {
			errReg := registrationImpl.Comms.SendNodeTopology(connectionID(nodeInfo.Id), &nodeTopology)
			if errReg != nil {
				return err
			}
		}
	}
	return nil
}

// getNodeInfo creates a NodeInfo mixmessage from the
// node info in the database and other input params
func getNodeInfo(dbNodeInfo *database.NodeInformation, index uint32, NodeTLSCert string) *mixmessages.NodeInfo {
	nodeInfo := mixmessages.NodeInfo{
		Id:        dbNodeInfo.Id,
		Index:     index,
		IpAddress: dbNodeInfo.Address,
		TlsCert:   NodeTLSCert,
	}

	return &nodeInfo
}

// outputDsaPubKeyToJson encodes the DSA public key to JSON and outputs it to
// the specified directory with the specified file name.
func outputDsaPubKeyToJson(publicKey *signature.DSAPublicKey, dir, fileName string) {
	// Encode the public key for the pem format
	encodedKey, err := publicKey.PemEncode()
	if err != nil {
		jww.ERROR.Printf("Error Pem encoding public key: %s", err)
	}

	// Setup struct that will dictate the JSON structure
	jsonStruct := struct {
		Dsa_public_key string
	}{
		Dsa_public_key: string(encodedKey),
	}

	// Generate JSON from structure
	data, err := json.MarshalIndent(jsonStruct, "", "\t")
	if err != nil {
		jww.ERROR.Printf("Error encoding structure to JSON: %s", err)
	}

	// Get the user's home directory
	homeDir, err := homedir.Dir()
	if err != nil {
		jww.ERROR.Printf("Unable to retrieve user's home directory: %s", err)
	}

	// Write JSON to file
	err = ioutil.WriteFile(homeDir+"/"+dir+"/"+fileName, data, 0644)
	if err != nil {
		jww.ERROR.Printf("Error writing JSON file: %s", err)
	}
}
