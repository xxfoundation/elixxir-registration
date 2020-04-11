////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/shuffle"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"strconv"
	"time"
)

func createRound(params Params, pool *waitingPoll, roundID *RoundID,
	updateID *UpdateID, state *storage.NetworkState) error {
	//get the nodes for the team
	nodes := pool.Clear()

	params.LastRound = time.Now()

	//build the topology
	nodeMap := state.GetNodeMap()
	nodeStateList := make([]*node.State, params.TeamSize)
	orderedNodeList := make([]*id.Node, params.TeamSize)

	randomIndex := make([]uint64, params.TeamSize)
	for i := range randomIndex {
		randomIndex[i] = uint64(i)
	}

	if params.RandomOrdering {
		//random order
		// https://en.wikipedia.org/wiki/Fisher%E2%80%93Yates_shuffle
		// already a crypto definition
		shuffle.Shuffle(&randomIndex)
		for i, nid := range nodes {
			n := nodeMap.GetNode(nid)
			nodeStateList[i] = n
			orderedNodeList[randomIndex[i]] = nid
		}
	} else {
		for i, nid := range nodes {
			n := nodeMap.GetNode(nid)
			nodeStateList[i] = n
			position, err := strconv.Atoi(n.GetOrdering())
			if err != nil {
				return errors.WithMessagef(err,
					"Could not parse ordering info ('%s') from node %s",
					n.GetOrdering(), nid)
			}

			orderedNodeList[position] = nid
		}
	}

	topology := connect.NewCircuit(orderedNodeList)

	//create the round
	r, err := state.GetRoundMap().AddRound(roundID.Next(), params.BatchSize, topology)

	if err != nil {
		fmt.Println("failed to add round to map")
		return errors.WithMessagef(err, "Failed to create new round %v", roundID.Get())
	}

	//move the round to precomputing
	err = r.Update(states.PRECOMPUTING, time.Now())
	if err != nil {
		return errors.WithMessagef(err, "Could not move new round into %s", states.PRECOMPUTING)
	}

	//tag all nodes to the round
	for i, n := range nodeStateList {
		err := n.SetRound(r)
		if err != nil {
			fmt.Println("failed to set round")
			return errors.WithMessagef(err, "could not add round %v to node %s", r.GetRoundID(), nodes[i])
		}
	}

	//issue the update
	err = state.AddRoundUpdate(updateID.Next(), r.BuildRoundInfo())
	if err != nil {
		fmt.Printf("failed to add round update")
		return errors.WithMessagef(err, "Could not issue "+
			"update to create round %v", r.GetRoundID())
	}

	return nil
}
