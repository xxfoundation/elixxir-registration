////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package scheduling

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
)

// createSimpleRound.go contains the logic to construct a team for a round and
// add that round to the network state

// createSimpleRound.go builds a team for a round out of a pool and round id and places
// this round into the network state
func createSimpleRound(params Params, pool *waitingPool, roundID id.Round,
	state *storage.NetworkState) (protoRound, error) {

	nodes, err := pool.PickNRandAtThreshold(int(params.TeamSize), int(params.TeamSize))

	if err != nil {
		return protoRound{}, errors.Errorf("Failed to pick random node group: %v", err)
	}

	var newRound protoRound

	//build the topology
	nodeStateList := make([]*node.State, params.TeamSize)
	orderedNodeList := make([]*id.ID, params.TeamSize)

	// Generate a team based on latency
	nodeStateList, err = generateSemiOptimalOrdering(nodes, state)
	if err != nil {
		return protoRound{}, errors.WithMessage(err,
			"Failed to generate optimal ordering")
	}

	// Parse the node list to get the order
	for i, n := range nodeStateList {
		nid := n.GetID()
		orderedNodeList[i] = nid
	}

	// Construct the proto-round object
	newRound.Topology = connect.NewCircuit(orderedNodeList)
	newRound.ID = roundID
	newRound.BatchSize = params.BatchSize
	newRound.NodeStateList = nodeStateList
	newRound.ResourceQueueTimeout = params.ResourceQueueTimeout
	return newRound, nil

}
