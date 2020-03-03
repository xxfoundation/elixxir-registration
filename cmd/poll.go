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
	"gitlab.com/elixxir/primitives/ndf"
)

// Server->Permissioning unified poll function
func (m *RegistrationImpl) Poll(msg *pb.PermissioningPoll,
	auth *connect.Auth) (*pb.PermissionPollResponse, error) {

	return nil, nil
}

//PollNdf handles the client polling for an updated NDF
func (m *RegistrationImpl) PollNdf(theirNdfHash []byte, auth *connect.Auth) ([]byte, error) {

	// Lock the reading of fullNdfHash and check if it's been writen to
	m.ndfLock.RLock()
	defer m.ndfLock.RUnlock()
	ndfHashLen := len(m.fullNdfHash)
	//Check that the registration server has built an NDF
	if ndfHashLen == 0 {
		errMsg := errors.Errorf(ndf.NO_NDF)
		jww.WARN.Printf(errMsg.Error())
		return nil, errMsg
	}

	// Handle client request
	if !auth.IsAuthenticated || auth.Sender.IsDynamicHost() {
		// Do not return NDF if client hash matches
		if bytes.Compare(m.partialNdfHash, theirNdfHash) == 0 {
			return nil, nil
		}

		// Send the json of the client
		jww.DEBUG.Printf("Returning a new NDF to client!")
		return m.partialNdf, nil
	}

	// Do not return NDF if backend hash matches
	if bytes.Compare(m.partialNdfHash, theirNdfHash) == 0 {
		return nil, nil
	}

	//Send the json of the ndf
	jww.DEBUG.Printf("Returning a new NDF to a back-end server!")
	return m.fullNdf, nil
}
