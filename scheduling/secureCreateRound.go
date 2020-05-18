package scheduling

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"math"
	"strconv"
	"time"
)

// createSimpleRound builds the team for a round of a pool and round id

func createSecureRound(params Params, pool *waitingPool, roundID id.Round,
	state *storage.NetworkState) (protoRound, error) {

	latencyMap := createLatencyTable()
	// fixme: have a less arbitrary timeout (possibly add to params?)
	pool.CleanOfflineNodes(params.NodeCleanUpInterval * time.Minute)

	// fixme: find an appropriate number for the threshold. For testability it needs to be programmable
	nodes, err := pool.PickNRandAtThreshold(int(params.Threshold), int(params.TeamSize))
	if err != nil {
		return protoRound{}, errors.Errorf("Failed to pick random node group: %v", err)
	}

	jww.TRACE.Printf("Beginning permutations")
	start := time.Now()
	// Make all permutations of nodes
	permutations := Permute(nodes)

	// This assumes order is geographic region, where (arbitrarily)
	// 0 - Western United States and Canada
	// 1 - Eastern United States and Canada, Latin America
	// 2 - Western Europe, Africa
	// 3 - Eastern Europe, Russia, Middle East
	// 4 - Asia
	// We shall assume geographical distance causes latency in a naive
	//  manner, as delineated here:
	//  https://docs.google.com/document/d/1oyjIDlqC54u_eoFzQP9SVNU2IqjnQOjpUYd9aqbg5X0/edit#

	jww.DEBUG.Printf("Looking for most efficient teaming order")

	bestTime := math.MaxInt32
	var bestOrder []*node.State
	// Fixme: way to do this more efficiently?
	for _, nodes := range permutations {
		totalLatency := 0
		for i := 0; i < len(nodes); i++ {
			// Get the ordering for the current node
			ourRegion, err := strconv.Atoi(nodes[i].GetOrdering())
			if err != nil {
				return protoRound{}, errors.WithMessagef(err,
					"Could not parse ordering info ('%s') from node %s",
					nodes[i].GetOrdering(), nodes[i].GetID())

			}

			// Get the ordering of the next node, circling back if at the last node
			nextNode := nodes[(i+1)%len(nodes)]
			nextRegion, err := strconv.Atoi(nextNode.GetOrdering())
			if err != nil {
				return protoRound{}, errors.WithMessagef(err,
					"Could not parse ordering info ('%s') from node %s",
					nextNode.GetOrdering(), nextNode.GetID())

			}

			// Calculate the distance and pull the latency from the table
			distance := Abs(nextRegion - ourRegion)
			totalLatency += latencyMap[distance]

			// Stop if this permutation has already accumulated
			// a latency worse than the best time
			//  and move to next iteration
			if totalLatency > bestTime {
				break
			}
		}

		// Replace with the best time and order found thus far
		if totalLatency < bestTime {
			bestOrder = nodes
			bestTime = totalLatency
		}

	}

	jww.DEBUG.Printf("Permute/Find best team took: %v", time.Now().Sub(start))

	// Create proto
	newRound := createProtoRound(params, state, bestOrder, roundID)

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

	return
}

// Creates a latency table mapping regional distance to latency
// todo: table needs better real-world accuracy
func createLatencyTable() (distanceLatency map[int]int) {
	distanceLatency = make(map[int]int)

	distanceLatency[0] = 1
	distanceLatency[1] = 3
	distanceLatency[2] = 7
	distanceLatency[3] = 15
	distanceLatency[4] = 31

	return
}

// Abs returns the absolute value of x. There is no
// builtin for abs of int type
// todo: Put this in primitives?
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
