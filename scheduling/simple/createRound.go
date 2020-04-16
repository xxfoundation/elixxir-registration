////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"strconv"
	"time"
)

// createRound.go contains the logic to construct a team for a round and
//  add that round to the network state

// createRound.go builds a team for a round out of a pool and round id and places
//  this round into the network state
func createRound(params Params, pool *waitingPoll, roundID *RoundID,
	state *storage.NetworkState) (*connect.Circuit, id.Round, []*node.State, []*id.Node, error) {
	//get the nodes for the team
	nodes := pool.Clear()
	params.LastRound = time.Now()

	//build the topology
	nodeMap := state.GetNodeMap()
	nodeStateList := make([]*node.State, params.TeamSize)
	orderedNodeList := make([]*id.Node, params.TeamSize)

	if params.RandomOrdering {

		// Input an incrementing array of ints
		randomIndex := make([]uint64, params.TeamSize)
		for i := range randomIndex {
			randomIndex[i] = uint64(i)
		}

		// Shuffle array of ints randomly using Fisher-Yates shuffle
		// https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
		shuffle.Shuffle(&randomIndex)
		for i, nid := range nodes {
			n := nodeMap.GetNode(nid)
			nodeStateList[i] = n
			// Use the shuffled array as an indexing order for
			//  the nodes' topological order
			orderedNodeList[randomIndex[i]] = nid
		}
	} else {
		for i, nid := range nodes {
			n := nodeMap.GetNode(nid)
			nodeStateList[i] = n
			position, err := strconv.Atoi(n.GetOrdering())
			if err != nil {
				return nil, id.Round(0), nil, nil, errors.WithMessagef(err,
					"Could not parse ordering info ('%s') from node %s",
					n.GetOrdering(), nid)
			}

			orderedNodeList[position] = nid
		}
	}

	topology := connect.NewCircuit(orderedNodeList)

	return topology, roundID.Next(), nodeStateList, nodes, nil

}
