////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating polling callbacks for hooking into comms library

package cmd

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/elixxir/registration/storage/node"
	"net"
	"sync/atomic"
	"time"
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

	fmt.Println("after start, node conn is at: ", n.GetConnectivity())

	// Check if the node has been deemed out of network
	if n.IsBanned() {
		return nil, errors.Errorf("Node %s has been banned from the network", nid)
	}

	//update ip addresses if nessessary
	err = checkIPAddresses(m, n, msg, serverAddress)
	if err != nil {
		err = errors.WithMessage(err, "Failed to update IP addresses")
		return
	}

	// Check for correct version
	err = checkVersion(m.params.minGatewayVersion, m.params.minServerVersion,
		msg)
	if err != nil {
		return nil, err
	}

	// Return updated NDF if provided hash does not match current NDF hash
	if isSame := m.State.GetFullNdf().CompareHash(msg.Full.Hash); !isSame {
		jww.TRACE.Printf("Returning a new NDF to a back-end server!")

		// Return the updated NDFs
		response.FullNDF = m.State.GetFullNdf().GetPb()
		response.PartialNDF = m.State.GetPartialNdf().GetPb()
	}

	// Fetch latest round updates
	response.Updates, err = m.State.GetUpdates(int(msg.LastUpdate))
	if err != nil {
		return
	}

	activity := current.Activity(msg.Activity)
	fmt.Println("connectivity: ", n.GetConnectivity())

	// Check the node's connectivity
	continuePoll, err := checkPortForwarding(n, activity, serverAddress)
	if err != nil || !continuePoll {
		return
	}

	// Increment the Node's poll count
	n.IncrementNumPolls()

	// Commit updates reported by the node if node involved in the current round
	jww.TRACE.Printf("Updating state for node %s: %+v",
		auth.Sender.GetId(), msg)

	//catch edge case with malformed error and return it to the node
	if current.Activity(msg.Activity) == current.ERROR && msg.Error == nil {
		err = errors.Errorf("A malformed error was received from %s "+
			"with a nil error payload", nid)
		jww.WARN.Println(err)
		return response, err
	}

	// if the node is in not started state, do not produce an update
	if activity == current.NOT_STARTED {
		return
	}

	// Return early before we get the polling lock if round creation stopped
	stopped := atomic.LoadUint32(m.Stopped) == 1
	if !stopped {
		// when a node poll is received, the nodes polling lock is taken here. If
		// there is no update, it is released in this endpoint, otherwise it is
		// released in the scheduling algorithm which blocks all future polls until
		// processing completes
		n.GetPollingLock().Lock()

		err = verifyError(msg, n, m)
		if err != nil {
			n.GetPollingLock().Unlock()
			return response, err
		}
	}

	// update does edge checking. It ensures the state change recieved was a
	// valid one and the state fo the node and
	// any associated round allows for that change. If the change was not
	// acceptable, it is not recorded and an error is returned, which is
	// propagated to the node
	update, updateNotification, err := n.Update(current.Activity(msg.Activity))
	if !stopped {
		//if updating to an error state, attach the error the the update
		if update && err == nil && updateNotification.ToActivity == current.ERROR {
			updateNotification.Error = msg.Error
		}

		//if an update ocured, report it to the control thread
		if update {
			err = m.State.SendUpdateNotification(updateNotification)
		} else {
			n.GetPollingLock().Unlock()
		}
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
		jww.TRACE.Printf("Returning a new NDF to client!")
		jww.TRACE.Printf("Sending the following ndf: %v", m.State.GetPartialNdf().Get())
		return m.State.GetPartialNdf().Get().Marshal()
	}

	// Do not return NDF if backend hash matches
	if isSame := m.State.GetFullNdf().CompareHash(theirNdfHash); isSame {
		return nil, nil
	}

	//Send the json of the ndf
	jww.TRACE.Printf("Returning a new NDF to a back-end server!")
	return m.State.GetFullNdf().Get().Marshal()
}

// checkVersion checks if the PermissioningPoll message server and gateway
// versions are compatible with the required version.
func checkVersion(requiredGateway, requiredServer version.Version,
	msg *pb.PermissioningPoll) error {

	// Skip checking gateway if the server is polled before gateway resulting in
	// a blank gateway version
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
				"the required version %#v.",
				gatewayVersion.String(), requiredGateway.String())
		}
	} else {
		jww.TRACE.Printf("Gateway version string is empty. Skipping gateway " +
			"version check.")
	}

	// Parse the server version string
	serverVersion, err := version.ParseVersion(msg.GetServerVersion())
	if err != nil {
		return errors.Errorf("Failed to parse server version %#v: %+v",
			msg.GetServerVersion(), err)
	}

	// Check that the server version is compatible with the required version
	if !version.IsCompatible(requiredServer, serverVersion) {
		return errors.Errorf("The server version %#v is incompatible with "+
			"the required version %#v.",
			serverVersion.String(), requiredServer.String())
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

// Verify that the error in permissioningpoll is valid
// Returns an error if invalid, or nil if valid or no error
func verifyError(msg *pb.PermissioningPoll, n *node.State, m *RegistrationImpl) error {
	// If there is an error, we must verify the signature before an update occurs
	// We do not want to update if the signature is invalid
	if msg.Error != nil {
		// only ensure there is an associated round if the error reports
		// association with a round
		if msg.Error.Id != 0 {
			ok, r := n.GetCurrentRound()
			if !ok {
				return errors.New("Node cannot submit a rounderror when it is not participating in a round")
			} else if msg.Error.Id != uint64(r.GetRoundID()) {
				return errors.New("This error is not associated with the round the submitting node is participating in")
			}
		}

		//check the error is signed by the node that created it
		errorNodeId, err := id.Unmarshal(msg.Error.NodeId)
		if err != nil {
			return errors.WithMessage(err, "Could not unmarshal node ID from error in poll")
		}
		h, ok := m.Comms.GetHost(errorNodeId)
		if !ok {
			return errors.Errorf("Host %+v was not found in host map", errorNodeId)
		}
		nodePK := h.GetPubKey()
		err = signature.Verify(msg.Error, nodePK)
		if err != nil {
			return errors.WithMessage(err, "Failed to verify error signature")
		}
	}
	return nil
}

func checkIPAddresses(m *RegistrationImpl, n *node.State, msg *pb.PermissioningPoll, nodeAddress string) error {
	// Get server and gateway addresses
	gatewayAddress := msg.GatewayAddress

	// Update server and gateway addresses in state, if necessary
	nodeUpdate := n.UpdateNodeAddresses(nodeAddress)
	gatewayUpdate := n.UpdateGatewayAddresses(gatewayAddress)

	var err error

	jww.TRACE.Printf("Received gateway and node update: %s, %s", nodeAddress,
		gatewayAddress)

	// If state required changes, then check the NDF
	if nodeUpdate || gatewayUpdate {

		jww.TRACE.Printf("UPDATING gateway and node update: %s, %s", nodeAddress,
			gatewayAddress)
		m.NDFLock.Lock()
		currentNDF := m.State.GetFullNdf().Get()

		if nodeUpdate {
			if err = updateNdfNodeAddr(n.GetID(), nodeAddress, currentNDF); err != nil {
				m.NDFLock.Unlock()
				return err
			}
		}
		if gatewayUpdate {
			if err = updateNdfGatewayAddr(n.GetID(), gatewayAddress, currentNDF); err != nil {
				m.NDFLock.Unlock()
				return err
			}
		}

		// Update the internal state with the newly-updated ndf
		if err = m.State.UpdateNdf(currentNDF); err != nil {
			m.NDFLock.Unlock()
			return err
		}
		m.NDFLock.Unlock()

		// Output the current topology to a JSON file
		err = outputToJSON(currentNDF, m.ndfOutputPath)
		if err != nil {
			err := errors.Errorf("unable to output NDF JSON file: %+v", err)
			jww.ERROR.Print(err.Error())
		}
	}

	return nil
}

// Handles the responses to the different connectivity states of a node
func checkPortForwarding(n *node.State, activity current.Activity, serverAddress string) (bool, error) {
	switch n.GetConnectivity() {
	case node.PortUnknown:
		// If we are not sure on whether the port has been forwarded
		go checkConnectivity(n, serverAddress)
		// Check that the node hasn't errored out
		if activity != current.ERROR {
			return true, nil
		}
	case node.PortVerifying:
		// If we are still verifying, then
		if activity != current.ERROR {
			return true, nil
		}
	case node.PortSuccessful:
		// In the case of a successful port check, we do nothing
		return true, nil
	case node.PortFailed:
		// If the port has been marked as failed,
		// we send an error informing the node of such
		return false, errors.Errorf("Node %s cannot be contacted "+
			"at %s by Permissioning, are ports properly forwarded?", n.GetID(),
			serverAddress)
	}

	return false, nil
}

// Attempts to dial node n at serverAddress.
// If we can successfully connect then, then port forwarding been done correctly
// Otherwise we return an error to the node
func checkConnectivity(n *node.State, serverAddress string) {
	// Then we ping the server and attempt on that port
	seconds := 5
	timeOut := time.Duration(seconds) * time.Second
	conn, errOpen := net.DialTimeout("tcp", serverAddress, timeOut)
	if errOpen != nil {
		// If we cannot connect, mark the node as failed
		jww.DEBUG.Printf("Failed to verify connectivity"+
			" for node (%s) at (%s): %s", n.GetID(), serverAddress, errOpen)
		n.SetConnectivity(node.PortFailed)
	} else {
		// If connection was successful, mark the port as forwarded
		n.SetConnectivity(node.PortSuccessful)

	}

	// Attempt to close the connection
	if conn != nil {
		errClose := conn.Close()
		if errClose != nil {
			jww.DEBUG.Printf("Failed to close connection for node (%s) at (%s): %v",
				n.GetID(), serverAddress, errClose)
		}
	}
}
