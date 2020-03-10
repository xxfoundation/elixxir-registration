////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating polling callbacks for hooking into comms library

package cmd

import (
	"errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
)

// Server->Permissioning unified poll function
func (m *RegistrationImpl) Poll(msg *pb.PermissioningPoll,
	auth *connect.Auth) (response *pb.PermissionPollResponse, err error) {

	// Initialize the response
	response = &pb.PermissionPollResponse{}

	// Ensure the NDF is ready to be returned
	if m.State.GetFullNdf() == nil || m.State.GetPartialNdf() == nil {
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
		response.FullNDF.Ndf, err = m.State.GetFullNdf().Get().Marshal()
		if err != nil {
			return
		}
		response.PartialNDF.Ndf, err = m.State.GetPartialNdf().Get().Marshal()
		if err != nil {
			return
		}

		// Sign the updated NDFs
		err = signature.Sign(response.FullNDF, m.State.PrivateKey)
		err = signature.Sign(response.PartialNDF, m.State.PrivateKey)
	}

	// Commit updates reported by the node if node involved in the current round
	if m.State.IsRoundNode(auth.Sender.GetId()) {
		jww.DEBUG.Printf("Updating state for node %s: %+v",
			auth.Sender.GetId(), msg)
		err = m.UpdateState(
			id.NewNodeFromBytes([]byte(auth.Sender.GetId())),
			(*current.Activity)(&msg.Activity))
		if err != nil {
			return
		}
	}

	// Fetch latest round updates
	response.Updates, err = m.State.GetUpdates(int(msg.LastUpdate))
	if err != nil {
		return
	}

	return
}

// PollNdf handles the client polling for an updated NDF
func (m *RegistrationImpl) PollNdf(theirNdfHash []byte, auth *connect.Auth) ([]byte, error) {

	// Ensure the NDF is ready to be returned
	if m.State.GetFullNdf() == nil || m.State.GetPartialNdf() == nil {
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
