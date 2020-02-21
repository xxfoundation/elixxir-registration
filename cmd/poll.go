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

// PollNdf handles the client polling for an updated NDF
func (m *RegistrationImpl) PollNdf(theirNdfHash []byte, auth *connect.Auth) ([]byte, error) {

	// Lock the reading of regNdfHash and check if it's been writen to
	m.ndfLock.RLock()
	defer m.ndfLock.RUnlock()
	ndfHashLen := len(m.regNdfHash)
	//Check that the registration server has built an NDF
	if ndfHashLen == 0 {
		return nil, errors.Errorf(ndf.NO_NDF)
	}

	//If both the sender's ndf hash and the permissioning NDF hash match
	//  no need to pass anything through the comm
	if bytes.Compare(m.regNdfHash, theirNdfHash) == 0 {
		return nil, nil
	}

	// Handle client request
	if !auth.IsAuthenticated || auth.Sender.IsDynamicHost() {
		jww.DEBUG.Printf("Returning a new NDF to client!")

		//Send the json of the client
		return m.clientNdf, nil

	}

	jww.DEBUG.Printf("Returning a new NDF to a back-end server!")
	//Send the json of the ndf
	return m.backEndNdf, nil
}
