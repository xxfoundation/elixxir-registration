////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating polling callbacks for hooking into comms library

package cmd

import (
	"bytes"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/version"
	"strconv"
	"strings"
	"sync/atomic"
)

// Server->Permissioning unified poll function
func (m *RegistrationImpl) Poll(msg *pb.PermissioningPoll, auth *connect.Auth,
	serverAddress string) (response *pb.PermissionPollResponse, err error) {

	// Initialize the response
	response = &pb.PermissionPollResponse{}

	// Ensure the NDF is ready to be returned
	regComplete := atomic.LoadUint32(m.NdfReady)
	if regComplete != 1 {
		return response, errors.New(ndf.NO_NDF)
	}

	// Ensure client is properly authenticated
	if !auth.IsAuthenticated || auth.Sender.IsDynamicHost() {
		return response, connect.AuthError(auth.Sender.GetId())
	}

	// Get the nodeState and update
	nid := auth.Sender.GetId()
	n := m.State.GetNodeMap().GetNode(nid)
	if n == nil {
		err = errors.Errorf("Node %s could not be found in internal state "+
			"tracker", nid)
		return
	}

	// Increment the Node's poll count
	n.IncrementNumPolls()

	// Check if the node has been deemed out of network
	if n.IsBanned() {
		return nil, errors.Errorf("Node %s has been banned from the network", nid)
	}

	// Check for correct version
	err = checkVersion(m.params.minGatewayVersion, m.params.minServerVersion,
		msg)
	if err != nil {
		return nil, err
	}

	// Get server and gateway addresses
	serverPortString := strconv.Itoa(int(msg.ServerPort))
	nodeAddress := strings.Join([]string{serverAddress, serverPortString}, ":")
	gatewayAddress := msg.GatewayAddress

	// Update server and gateway addresses in state, if necessary
	nodeUpdate := n.UpdateNodeAddresses(nodeAddress)
	gatewayUpdate := n.UpdateGatewayAddresses(gatewayAddress)

	// If state required changes, then check the NDF
	if nodeUpdate || gatewayUpdate {
		currentNDF := m.State.GetFullNdf().Get()

		if nodeUpdate {
			if err = updateNdfNodeAddr(nid, serverAddress, currentNDF); err != nil {
				return
			}
		}
		if gatewayUpdate {
			if err = updateNdfGatewayAddr(nid, gatewayAddress, currentNDF); err != nil {
				return
			}
		}

		// Update the internal state with the newly-updated ndf
		if err = m.State.UpdateNdf(currentNDF); err != nil {
			return
		}
	}

	// Return updated NDF if provided hash does not match current NDF hash
	if isSame := m.State.GetFullNdf().CompareHash(msg.Full.Hash); !isSame {
		jww.DEBUG.Printf("Returning a new NDF to a back-end server!")

		// Return the updated NDFs
		response.FullNDF = m.State.GetFullNdf().GetPb()
		response.PartialNDF = m.State.GetPartialNdf().GetPb()
	}

	// Fetch latest round updates
	response.Updates, err = m.State.GetUpdates(int(msg.LastUpdate))
	if err != nil {
		return
	}

	// Commit updates reported by the node if node involved in the current round
	jww.DEBUG.Printf("Updating state for node %s: %+v",
		auth.Sender.GetId(), msg)

	// when a node poll is received, the nodes polling lock is taken here. If
	// there is no update, it is released in this endpoint, otherwise it is
	// released in the scheduling algorithm which blocks all future polls until
	// processing completes
	n.GetPollingLock().Lock()

	// update does edge checking. It ensures the state change recieved was a
	// valid one and the state fo the node and
	// any associated round allows for that change. If the change was not
	// acceptable, it is not recorded and an error is returned, which is
	// propagated to the node
	update, updateNotification, err := n.Update(current.Activity(msg.Activity))

	//if an update ocured, report it to the control thread
	if update {
		err = m.State.SendUpdateNotification(updateNotification)
	} else {
		n.GetPollingLock().Unlock()
	}

	return
}

// PollNdf handles the client polling for an updated NDF
func (m *RegistrationImpl) PollNdf(theirNdfHash []byte, auth *connect.Auth) ([]byte, error) {

	// Ensure the NDF is ready to be returned
	regComplete := atomic.LoadUint32(m.NdfReady)
	if regComplete != 1 {
		return nil, errors.New(ndf.NO_NDF)
	}

	// Handle client request
	if !auth.IsAuthenticated || auth.Sender.IsDynamicHost() {
		// Do not return NDF if client hash matches
		if isSame := m.State.GetPartialNdf().CompareHash(theirNdfHash); isSame {
			return nil, nil
		}

		// Send the json of the client
		jww.DEBUG.Printf("Returning a new NDF to client!")
		return m.State.GetPartialNdf().Get().Marshal()
	}

	// Do not return NDF if backend hash matches
	if isSame := m.State.GetFullNdf().CompareHash(theirNdfHash); isSame {
		return nil, nil
	}

	//Send the json of the ndf
	jww.DEBUG.Printf("Returning a new NDF to a back-end server!")
	return m.State.GetFullNdf().Get().Marshal()
}

// checkVersion checks if the PermissioningPoll message server and gateway
// versions are compatible with the required version.
func checkVersion(requiredGateway, requiredServer version.Version,
	msg *pb.PermissioningPoll) error {

	if msg.GetGatewayVersion() != "" {
		// Parse the gateway version string
		gatewayVersion, err := version.ParseVersion(msg.GetGatewayVersion())
		if err != nil {
			return errors.Errorf("Failed to parse gateway version %#v: %+v",
				msg.GetGatewayVersion(), err)
		}

		// Check that the gateway version is compatible with the required version
		if !version.IsCompatible(requiredGateway, gatewayVersion) {
			return errors.Errorf("The gateway version %#v is incompatible with "+
				"the required version %#v.", gatewayVersion.String(), requiredGateway.String())
		}
	}

	if msg.GetServerVersion() != "" {
		// Parse the server version string
		serverVersion, err := version.ParseVersion(msg.GetServerVersion())
		if err != nil {
			return errors.Errorf("Failed to parse server version %#v: %+v",
				msg.GetServerVersion(), err)
		}

		// Check that the server version is compatible with the required version
		if !version.IsCompatible(requiredServer, serverVersion) {
			return errors.Errorf("The server version %#v is incompatible with "+
				"the required version %#v.", serverVersion.String(), requiredServer.String())
		}
	}

	return nil
}

// updateNdfNodeAddr searches the NDF nodes for a matching node ID and updates
// its address to the required address.
func updateNdfNodeAddr(nid *id.ID, requiredAddr string, ndf *ndf.NetworkDefinition) error {
	replaced := false

	// TODO: Have a faster search with an efficiency greater than O(n)
	// Search the list of NDF nodes for a matching ID and update the address
	for i, n := range ndf.Nodes {
		if bytes.Equal(n.ID, nid[:]) {
			ndf.Nodes[i].Address = requiredAddr
			replaced = true
			break
		}
	}

	// Return an error if no matching node is found
	if !replaced {
		return errors.Errorf("Could not find node %s in the state map in "+
			"order to update its address", nid.String())
	}

	return nil
}

// updateNdfGatewayAddr searches the NDF gateways for a matching gateway ID and
// updates its address to the required address.
func updateNdfGatewayAddr(nid *id.ID, requiredAddr string, ndf *ndf.NetworkDefinition) error {
	replaced := false
	gid := nid.DeepCopy()
	gid.SetType(id.Gateway)

	// TODO: Have a faster search with an efficiency greater than O(n)
	// Search the list of NDF gateways for a matching ID and update the address
	for i, gw := range ndf.Gateways {
		if bytes.Equal(gw.ID, gid[:]) {
			ndf.Gateways[i].Address = requiredAddr
			replaced = true
			break
		}
	}

	// Return an error if no matching gateway is found
	if !replaced {
		return errors.Errorf("Could not find gateway %s in the state map "+
			"in order to update its address", gid.String())
	}

	return nil
}
