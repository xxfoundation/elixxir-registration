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
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/comms/signature"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/ndf"
	"gitlab.com/xx_network/primitives/utils"
	"math/rand"
	"sync/atomic"
)

// Server->Permissioning unified poll function
func (m *RegistrationImpl) Poll(msg *pb.PermissioningPoll, auth *connect.Auth) (*pb.PermissionPollResponse, error) {

	// Initialize the response
	response := &pb.PermissionPollResponse{}

	//do edge check to ensure the message is not nil
	if msg == nil {
		return nil, errors.Errorf("Message payload for unified poll " +
			"is nil, poll cannot be processed")
	}

	// Ensure client is properly authenticated
	if !auth.IsAuthenticated || auth.Sender.IsDynamicHost() {
		return response, connect.AuthError(auth.Sender.GetId())
	}

	// Check for correct version
	err := checkVersion(m.params, msg)
	if err != nil {
		return response, err
	}

	// Get the nodeState and update
	nid := auth.Sender.GetId()
	n := m.State.GetNodeMap().GetNode(nid)
	if n == nil {
		err = errors.Errorf("Node %s could not be found in internal state "+
			"tracker", nid)
		return response, err
	}

	// Check if the node has been deemed out of network
	if n.IsBanned() {
		return response, errors.Errorf("Node %s has been banned from the network", nid)
	}

	activity := current.Activity(msg.Activity)

	// update ip addresses if necessary
	err = checkIPAddresses(m, n, msg, auth.Sender)
	if err != nil {
		err = errors.WithMessage(err, "Failed to update IP addresses")
		return response, err
	}

	// Check the node's connectivity
	continuePoll, err := m.checkConnectivity(n, auth.IpAddress, activity,
		m.GetDisableGatewayPingFlag())
	if err != nil || !continuePoll {
		return response, err
	}

	// Increment the Node's poll count
	n.IncrementNumPolls()

	// Ensure the NDF is ready to be returned
	regComplete := atomic.LoadUint32(m.NdfReady)
	if regComplete != 1 {
		return response, errors.New(ndf.NO_NDF)
	}

	// Return updated NDF if provided hash does not match current NDF hash
	if isSame := m.State.GetFullNdf().CompareHash(msg.Full.Hash); !isSame {
		jww.TRACE.Printf("Returning a new NDF to a back-end server!")

		// Return the updated NDFs
		response.FullNDF = m.State.GetFullNdf().GetPb()
		response.PartialNDF = m.State.GetPartialNdf().GetPb()
	} else {
		// Fetch latest round updates
		response.Updates, err = m.State.GetUpdates(int(msg.LastUpdate))
		if err != nil {
			return response, err
		}
	}

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

	// If round creation stopped OR if the node is in not started state,
	// return early before we get the polling lock
	stopped := atomic.LoadUint32(m.Stopped) == 1
	if activity == current.NOT_STARTED || stopped {
		return response, err
	}

	// Ensure any errors are properly formatted before sending an update
	err = verifyError(msg, n, m)
	if err != nil {
		return response, err
	}

	//check if the node is pruned if it is, bail
	if m.State.IsPruned(n.GetID()) {
		return response, err
	}

	// when a node poll is received, the nodes polling lock is taken here. If
	// there is no update, it is released in this endpoint, otherwise it is
	// released in the scheduling algorithm which blocks all future polls until
	// processing completes
	n.GetPollingLock().Lock()

	// update does edge checking. It ensures the state change received was a
	// valid one and the state of the node and
	// any associated round allows for that change. If the change was not
	// acceptable, it is not recorded and an error is returned, which is
	// propagated to the node
	isUpdate, updateNotification, err := n.Update(current.Activity(msg.Activity))
	if !isUpdate || err != nil {
		n.GetPollingLock().Unlock()
		return response, err
	}

	// If updating to an error state, attach the error the the update
	if updateNotification.ToActivity == current.ERROR {
		updateNotification.Error = msg.Error
	}
	updateNotification.ClientErrors = msg.ClientErrors

	// Update occurred, report it to the control thread
	return response, m.State.SendUpdateNotification(updateNotification)
}

// PollNdf handles the client polling for an updated NDF
func (m *RegistrationImpl) PollNdf(theirNdfHash []byte) (*pb.NDF, error) {

	// Ensure the NDF is ready to be returned
	regComplete := atomic.LoadUint32(m.NdfReady)
	if regComplete != 1 {
		return nil, errors.New(ndf.NO_NDF)
	}

	// Do not return NDF if backend hash matches
	if isSame := m.State.GetPartialNdf().CompareHash(theirNdfHash); isSame {
		return &pb.NDF{}, nil
	}

	//Send the json of the ndf
	jww.TRACE.Printf("Returning a new NDF to a back-end server!")
	return m.State.GetPartialNdf().GetPb(), nil
}

// checkVersion checks if the PermissioningPoll message server and gateway
// versions are compatible with the required version.
func checkVersion(p *Params, msg *pb.PermissioningPoll) error {

	// Pull the versions
	p.versionLock.RLock()
	requiredGateway := p.minGatewayVersion
	requiredServer := p.minServerVersion
	p.versionLock.RUnlock()

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
		err = signature.VerifyRsa(msg.Error, nodePK)
		if err != nil {
			return errors.WithMessage(err, "Failed to verify error signature")
		}
	}
	return nil
}

func checkIPAddresses(m *RegistrationImpl, n *node.State,
	msg *pb.PermissioningPoll, nodeHost *connect.Host) error {

	// Pull the addresses out of the message
	gatewayAddress, nodeAddress := msg.GatewayAddress, msg.ServerAddress

	// Update server and gateway addresses in state, if necessary
	nodeUpdate := n.UpdateNodeAddresses(nodeAddress)
	gatewayUpdate := n.UpdateGatewayAddresses(gatewayAddress)

	// If state required changes, then check the NDF
	if nodeUpdate || gatewayUpdate {

		jww.TRACE.Printf("UPDATING gateway and node update: %s, %s", msg.ServerAddress,
			gatewayAddress)

		// Update address information in Storage
		err := storage.PermissioningDb.UpdateNodeAddresses(nodeHost.GetId(), nodeAddress, gatewayAddress)
		if err != nil {
			return err
		}

		m.NDFLock.Lock()
		currentNDF := m.State.GetUnprunedNdf()

		n.SetConnectivity(node.PortUnknown)

		if nodeUpdate {
			nodeHost.UpdateAddress(nodeAddress)
			if err := updateNdfNodeAddr(n.GetID(), nodeAddress, currentNDF); err != nil {
				m.NDFLock.Unlock()
				return err
			}
		}

		if gatewayUpdate {
			if err := updateNdfGatewayAddr(n.GetID(), gatewayAddress, currentNDF); err != nil {
				m.NDFLock.Unlock()
				return err
			}
		}

		// Update the internal state with the newly-updated ndf
		if err := m.State.UpdateNdf(currentNDF); err != nil {
			m.NDFLock.Unlock()
			return err
		}
		m.NDFLock.Unlock()
	}

	return nil
}

// checkConnectivity handles the responses to the different connectivity states
// of a node. If the returned boolean is true, then the poll should continue.
// The nodeIpAddr is the IP of of the node when it connects to permissioning; it
// is not the IP or domain name reported by the node.
func (m *RegistrationImpl) checkConnectivity(n *node.State, nodeIpAddr string,
	activity current.Activity, disableGatewayPing bool) (bool, error) {

	switch n.GetConnectivity() {
	case node.PortUnknown:
		err := m.setNodeSequence(n, nodeIpAddr)
		if err != nil {
			return false, err
		}
		// If we are not sure on whether the port has been forwarded
		// Ping the server and attempt on that port
		go func() {
			nodeHost, exists := m.Comms.GetHost(n.GetID())
			jww.INFO.Printf("[CHECKCONN]Node %s address in host object: %s", nodeHost.GetId(), nodeHost.GetAddress())
			jww.INFO.Printf("[CHECKCONN]Node %s address IsPublicAddress: %s", nodeHost.GetId(), utils.IsPublicAddress(nodeHost.GetAddress()))
			jww.INFO.Printf("[CHECKCONN]Node %s address IsOnline: %t", nodeHost.GetId(), nodeHost.IsOnline())

			nodePing := exists &&
				utils.IsPublicAddress(nodeHost.GetAddress()) == nil &&
				nodeHost.IsOnline()

			gwPing := true
			if !disableGatewayPing {
				gwID := nodeHost.GetId().DeepCopy()
				gwID.SetType(id.Gateway)
				params := connect.GetDefaultHostParams()
				params.AuthEnabled = false
				gwHost, err := connect.NewHost(gwID, n.GetGatewayAddress(), nil, params)
				jww.INFO.Printf("[CHECKCONN]Gw %s address in host object: %s", nodeHost.GetId(), gwHost.GetAddress())
				jww.INFO.Printf("[CHECKCONN]Gw %s address IsPublicAddress: %s", nodeHost.GetId(), utils.IsPublicAddress(gwHost.GetAddress()))
				jww.INFO.Printf("[CHECKCONN]Gw %s address IsOnline: %t", nodeHost.GetId(), gwHost.IsOnline())
				gwPing = err == nil && utils.IsPublicAddress(n.GetGatewayAddress()) == nil && gwHost.IsOnline()
			}

			if nodePing && gwPing {
				// If connection was successful, mark the port as forwarded
				n.SetConnectivity(node.PortSuccessful)
			} else if !nodePing && gwPing {
				// If connection to Gateway was successful but Node was not
				n.SetConnectivity(node.NodePortFailed)
			} else if nodePing && !gwPing {
				// If connection to Node was successful but Gateway was not
				n.SetConnectivity(node.GatewayPortFailed)
			} else {
				// If we cannot connect to either address, mark the node as failed
				n.SetConnectivity(node.PortFailed)
			}
		}()
		// Check that the node hasn't errored out
		if activity == current.ERROR {
			return true, nil
		}

	case node.PortVerifying:
		// If we are still verifying, then
		if activity == current.ERROR {
			return true, nil
		}
	case node.PortSuccessful:
		// In the case of a successful port check for both Node and Gateway, we
		// do nothing
		return true, nil
	case node.NodePortFailed:

		// this will approximately force a recheck of the node state every 3~5
		// minutes
		if rand.Uint64()%211 == 13 {
			n.SetConnectivity(node.PortUnknown)
		}
		nodeAddress := "unknown"
		if nodeHost, exists := m.Comms.GetHost(n.GetID()); exists {
			nodeAddress = nodeHost.GetAddress()
		}
		// If only the Node port has been marked as failed,
		// we send an error informing the node of such
		return false, errors.Errorf("Node %s at %s cannot be contacted "+
			"by Permissioning, are ports properly forwarded?", n.GetID(), nodeAddress)
	case node.GatewayPortFailed:
		// this will approximately force a recheck of the node state every 3~5
		// minutes
		if rand.Uint64()%211 == 13 {
			n.SetConnectivity(node.PortUnknown)
		}
		gwID := n.GetID().DeepCopy()
		gwID.SetType(id.Gateway)
		// If only the Gateway port has been marked as failed,
		// we send an error informing the node of such
		return false, errors.Errorf("Gateway %s with address %s cannot be contacted "+
			"by Permissioning, are ports properly forwarded?", gwID, n.GetGatewayAddress())
	case node.PortFailed:
		// this will approximately force a recheck of the node state every 3~5
		// minutes
		if rand.Uint64()%211 == 13 {
			n.SetConnectivity(node.PortUnknown)
		}
		nodeAddress := "unknown"
		if nodeHost, exists := m.Comms.GetHost(n.GetID()); exists {
			nodeAddress = nodeHost.GetAddress()
		}
		// If the port has been marked as failed,
		// we send an error informing the node of such
		return false, errors.Errorf("Both Node %s at %s and Gateway with address %s "+
			"cannot be contacted by Permissioning, are ports properly forwarded?",
			n.GetID(), nodeAddress, n.GetGatewayAddress())
	}

	return false, nil
}
