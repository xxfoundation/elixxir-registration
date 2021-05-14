////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package round

import pb "gitlab.com/elixxir/comms/mixmessages"

//provides a utility function for making deep copies of roundInfo objects
func CopyRoundInfo(ri *pb.RoundInfo) *pb.RoundInfo {
	//copy the topology
	topology := ri.GetTopology()

	topologyCopy := make([][]byte, len(topology))
	for i, nid := range topology {
		topologyCopy[i] = make([]byte, len(nid))
		copy(topologyCopy[i], nid)
	}

	//copy the timestamps
	timestamps := ri.GetTimestamps()
	timestampsCopy := make([]uint64, len(timestamps))
	for i, stamp := range timestamps {
		timestampsCopy[i] = stamp
	}

	//copy the errors
	var errorsCopy []*pb.RoundError
	if len(ri.Errors) > 0 {
		errorsCopy = make([]*pb.RoundError, len(ri.Errors))
		for i, e := range ri.Errors {
			eCopy := *e
			if e.Signature != nil {
				sig := *(e.Signature)
				eCopy.Signature = &sig
			}
			errorsCopy[i] = &eCopy
		}
	}

	clientErrors := make([]*pb.ClientError, len(ri.ClientErrors))
	for i, err := range ri.ClientErrors {
		clientErrors[i] = &pb.ClientError{
			ClientId: make([]byte, len(err.ClientId)),
			Error:    err.Error,
			Source:   make([]byte, len(err.Source)),
		}
		copy(clientErrors[i].ClientId, err.ClientId)
		copy(clientErrors[i].Source, err.Source)
	}
	return &pb.RoundInfo{
		ID:                         ri.GetID(),
		State:                      ri.State,
		BatchSize:                  ri.GetBatchSize(),
		Topology:                   topologyCopy,
		Timestamps:                 timestampsCopy,
		Errors:                     errorsCopy,
		ClientErrors:               clientErrors,
		ResourceQueueTimeoutMillis: ri.GetResourceQueueTimeoutMillis(),
		AddressSpaceSize:           ri.GetAddressSpaceSize(),
	}
}
