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
	"gitlab.com/xx_network/primitives/region"
	"io"
	"time"
)

// createSimpleRound.go contains the logic to construct a team for a round and
// add that round to the network state

// createSimpleRound.go builds a team for a round out of a pool and round id and places
// this round into the network state
func createSimpleRound(params Params, pool *waitingPool, roundID id.Round,
	state *storage.NetworkState, rng io.Reader) (protoRound, error) {

	nodes, err := pool.PickNRandAtThreshold(int(params.TeamSize), int(params.TeamSize))

	if err != nil {
		return protoRound{}, errors.Errorf("Failed to pick random node group: %v", err)
	}

	var newRound protoRound

	//build the topology

	nodeIds := make([]*id.ID, 0, len(nodes))
	countries := make(map[id.ID]string)
	for _, n := range nodes {
		nodeIds = append(nodeIds, n.GetID())
		countries[*n.GetID()] = n.GetOrdering()
	}

	// Generate a team based on latency
	bestOrder, _, err := region.OrderNodeTeam(nodeIds, countries, region.GetCountryBins(),
		region.CreateSetLatencyTableWeights(region.CreateLinkTable()), rng)
	if err != nil {
		return protoRound{}, errors.WithMessage(err,
			"Failed to generate optimal ordering")
	}

	// Parse the node list to get the order
	nodeStateList := make([]*node.State, 0, params.TeamSize)
	for _, n := range bestOrder {
		nodeStateList = append(nodeStateList, state.GetNodeMap().GetNode(n))
	}

	// Construct the proto-round object
	newRound.Topology = connect.NewCircuit(bestOrder)
	newRound.ID = roundID
	newRound.BatchSize = params.BatchSize
	newRound.NodeStateList = nodeStateList
	newRound.ResourceQueueTimeout = params.ResourceQueueTimeout * time.Millisecond
	return newRound, nil

}
