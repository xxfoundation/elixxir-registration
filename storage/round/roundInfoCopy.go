////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package round

import (
	pb "git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/xx_network/comms/messages"
)

// TODO: add test
// CopyRoundInfo returns a deep copy of mixmessages.RoundInfo.
func CopyRoundInfo(ri *pb.RoundInfo) *pb.RoundInfo {
	// Copy the topology
	topologyCopy := make([][]byte, len(ri.GetTopology()))
	for i, nid := range ri.GetTopology() {
		topologyCopy[i] = make([]byte, len(nid))
		copy(topologyCopy[i], nid)
	}
	// Copy the timestamps
	timestampsCopy := make([]uint64, len(ri.GetTimestamps()))
	for i, stamp := range ri.GetTimestamps() {
		timestampsCopy[i] = stamp
	}
	// Copy the errors
	errorsCopy := make([]*pb.RoundError, len(ri.GetErrors()))
	for i, err := range ri.GetErrors() {
		errorsCopy[i] = &pb.RoundError{
			Id:     err.GetId(),
			NodeId: make([]byte, len(err.GetNodeId())),
			Error:  err.GetError(),
			Signature: &messages.RSASignature{
				Nonce:     make([]byte, len(err.GetSignature().GetNonce())),
				Signature: make([]byte, len(err.GetSignature().GetSignature())),
			},
		}
		copy(errorsCopy[i].NodeId, err.GetNodeId())
		copy(errorsCopy[i].Signature.Nonce, err.GetSignature().GetNonce())
		copy(errorsCopy[i].Signature.Signature, err.GetSignature().GetSignature())
	}
	clientErrors := make([]*pb.ClientError, len(ri.ClientErrors))
	for i, err := range ri.GetClientErrors() {
		clientErrors[i] = &pb.ClientError{
			ClientId: make([]byte, len(err.GetClientId())),
			Error:    err.GetError(),
			Source:   make([]byte, len(err.GetSource())),
		}
		copy(clientErrors[i].ClientId, err.GetClientId())
		copy(clientErrors[i].Source, err.GetSource())
	}
	signatureCopy := &messages.RSASignature{
		Nonce:     make([]byte, len(ri.GetSignature().GetNonce())),
		Signature: make([]byte, len(ri.GetSignature().GetSignature())),
	}
	copy(signatureCopy.Nonce, ri.GetSignature().GetNonce())
	copy(signatureCopy.Signature, ri.GetSignature().GetSignature())
	return &pb.RoundInfo{
		ID:                         ri.GetID(),
		UpdateID:                   ri.GetUpdateID(),
		State:                      ri.GetState(),
		BatchSize:                  ri.GetBatchSize(),
		Topology:                   topologyCopy,
		Timestamps:                 timestampsCopy,
		Errors:                     errorsCopy,
		ClientErrors:               clientErrors,
		ResourceQueueTimeoutMillis: ri.GetResourceQueueTimeoutMillis(),
		Signature:                  signatureCopy,
		AddressSpaceSize:           ri.GetAddressSpaceSize(),
	}
}
