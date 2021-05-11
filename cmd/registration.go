////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles creating client registration callbacks for hooking into comms library

package cmd

import (
	"crypto/rand"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/registration"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/xx_network/comms/messages"
	"time"
)

var rateLimitErr = errors.New("Too many client registrations. Try again later")

// Handle registration attempt by a Client
// Returns rsa signature and error
func (m *RegistrationImpl) RegisterUser(msg *pb.UserRegistration) (*pb.UserRegistrationConfirmation, error) {
	// Obtain the signed key by passing to registration server
	pubKey := msg.GetClientRSAPubKey()
	receptionKey := msg.GetClientReceptionRSAPubKey()
	regCode := msg.GetRegistrationCode()

	// Check for pre-existing registration for this public key first
	if user, err := storage.PermissioningDb.GetUser(pubKey); err == nil && user != nil {
		jww.WARN.Printf("Previous registration found for %s", pubKey)
	} else if regCode != "" {
		// Fail early for non-valid reg codes
		err = storage.PermissioningDb.UseCode(regCode)
		if err != nil {
			jww.WARN.Printf("RegisterUser error: %+v", err)
			return &pb.UserRegistrationConfirmation{}, err
		}
	} else if regCode == "" && !m.registrationLimiting.Add(1) {
		// Rate limited, fail early
		jww.WARN.Printf("RegisterUser error: %+v", rateLimitErr)
		return &pb.UserRegistrationConfirmation{}, rateLimitErr
	}

	// Sign the user's transmission and reception key with the time the user's registration was received
	now := time.Now()
	transmissionSig, err := registration.SignWithTimestamp(rand.Reader, m.State.GetPrivateKey(), now.UnixNano(), pubKey)
	if err != nil {
		jww.WARN.Printf("RegisterUser error: can't sign pubkey")
		return &pb.UserRegistrationConfirmation{}, errors.Errorf(
			"Unable to sign client public key: %+v", err)
	}

	receptionSig, err := registration.SignWithTimestamp(rand.Reader, m.State.GetPrivateKey(), now.UnixNano(), receptionKey)
	if err != nil {
		jww.WARN.Printf("RegisterUser error: can't sign receptionKey")
		return &pb.UserRegistrationConfirmation{}, errors.Errorf(
			"Unable to sign client reception key: %+v", err)
	}

	// Record the user public key for duplicate registration support
	err = storage.PermissioningDb.InsertUser(pubKey, receptionKey)
	if err != nil {
		jww.WARN.Printf("Unable to store user: %+v",
			errors.New(err.Error()))
	}

	// Return signed public key to Client
	jww.DEBUG.Printf("RegisterUser for code [%s] complete!", regCode)

	return &pb.UserRegistrationConfirmation{
		ClientSignedByServer: &messages.RSASignature{
			Signature: transmissionSig,
		},
		ClientReceptionSignedByServer: &messages.RSASignature{
			Signature: receptionSig,
		},
		Timestamp: now.UnixNano(),
	}, err
}
