package round

import pb "gitlab.com/elixxir/comms/mixmessages"

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

	return &pb.RoundInfo{
		ID:                         ri.GetID(),
		State:                      ri.State,
		BatchSize:                  ri.GetBatchSize(),
		Topology:                   topologyCopy,
		Timestamps:                 timestampsCopy,
		ResourceQueueTimeoutMillis: ri.GetResourceQueueTimeoutMillis(),
		Errors:                     errorsCopy,
	}
}
