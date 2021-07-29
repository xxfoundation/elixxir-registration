package scheduling

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// createSecureRound.go contains the logic to construct a team for a secure
// teaming algorithm. Focuses largely on constructing an optimal team

// createSimpleRound builds the team for a round of a pool and round id
// This for this we use the node state's order as its
// geographic region, where:
//    Americas       - Entirety of North and South America
//    Western Europe - todo define countries in region
//    Central Europe - todo define countries in region
//    Eastern Europe - todo define countries in region
//    Middle East    - todo define countries in region
//    Africa         - Consists of entire continent of Africa
//    Russia         - Consists of the country of Russia
//    Asia           - todo define countries in region
// We shall assume geographical distance causes latency in a naive
//  manner, as delineated here:
//  https://docs.google.com/document/d/1oyjIDlqC54u_eoFzQP9SVNU2IqjnQOjpUYd9aqbg5X0/edit#
func createSecureRound(params Params, pool *waitingPool, roundID id.Round,
	state *storage.NetworkState) (protoRound, error) {

	// Pick nodes from the pool
	nodes, err := pool.PickNRandAtThreshold(int(params.Threshold), int(params.TeamSize))
	if err != nil {
		return protoRound{}, errors.Errorf("Failed to pick random node group: %v", err)
	}

	jww.TRACE.Printf("Beginning permutations")
	start := time.Now()

	optimalTeam, err := generateSemiOptimalOrdering(nodes, state)
	if err != nil {
		return protoRound{}, errors.WithMessage(err,
			"Failed to generate optimal ordering")
	}

	jww.DEBUG.Printf("Permuting and finding the best team took: %v", time.Now().Sub(start))

	// Create proto-round object now that the optimal team has been found
	newRound := createProtoRound(params, state, optimalTeam, roundID)

	jww.TRACE.Printf("Built round %d", roundID)
	return newRound, nil
}

// CreateProtoRound is a helper function which creates a protoround object
func createProtoRound(params Params, state *storage.NetworkState,
	bestOrder []*node.State, roundID id.Round) (newRound protoRound) {

	// Pull information from the best order into a nodeStateList
	nodeIds := make([]*id.ID, params.TeamSize)
	nodeStateList := make([]*node.State, params.TeamSize)
	nodeMap := state.GetNodeMap()

	// Pull node id's out of the bestOrder list in order to make
	// a topology for the round
	for i := range bestOrder {
		nid := bestOrder[i].GetID()
		nodeIds[i] = nid
		n := nodeMap.GetNode(nid)
		nodeStateList[i] = n
	}

	// Build the protoRound
	newRound.Topology = connect.NewCircuit(nodeIds)
	newRound.ID = roundID
	newRound.BatchSize = params.BatchSize
	newRound.NodeStateList = nodeStateList
	newRound.ResourceQueueTimeout = params.ResourceQueueTimeout

	return
}
