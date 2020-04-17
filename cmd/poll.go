////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating polling callbacks for hooking into comms library

package cmd

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"sync/atomic"
)

// Server->Permissioning unified poll function
func (m *RegistrationImpl) Poll(msg *pb.PermissioningPoll,
	auth *connect.Auth) (response *pb.PermissionPollResponse, err error) {

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

	//get the nodeState and update
	nid, err := id.NewNodeFromString(auth.Sender.GetId())
	if err != nil {
		err = errors.Errorf("could not decode node id of %s: %s", auth.Sender.GetId(), err)
		return
	}
	n := m.State.GetNodeMap().GetNode(nid)
	if n == nil {
		err = errors.Errorf("node %s could not be found in internal state tracker", nid)
		return
	}

	update, oldActivity, err := n.Update(current.Activity(msg.Activity))

	//if an update ocured, report it to the control thread
	if update {
		err = m.State.NodeUpdateNotification(nid, oldActivity, current.Activity(msg.Activity))
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
