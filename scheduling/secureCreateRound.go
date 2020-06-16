package scheduling

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"math"
	"time"
)

// createSecureeRound.go contains the logic to construct a team for a secure
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

	// Create a latencyTable
	latencyMap := createLatencyTable()

	// Pick nodes from the pool
	nodes, err := pool.PickNRandAtThreshold(int(params.Threshold), int(params.TeamSize))
	if err != nil {
		return protoRound{}, errors.Errorf("Failed to pick random node group: %v", err)
	}

	jww.TRACE.Printf("Beginning permutations")
	start := time.Now()

	// Make all permutations of nodes
	permutations := Permute(nodes)

	jww.DEBUG.Printf("Looking for most efficient teaming order")
	optimalLatency := math.MaxInt32
	var optimalTeam []*node.State
	// TODO: consider a way to do this more efficiently? As of now,
	//  for larger teams of 10 or greater it takes >2 seconds for round creation
	//  but it runs in the microsecond range with 4 nodes.
	//  Since our use case is smaller teams, we deem this sufficient for now
	for _, nodes := range permutations {
		totalLatency := 0
		for i := 0; i < len(nodes); i++ {
			// Get the ordering for the current node
			thisRegion, err := getRegion(nodes[i].GetOrdering())
			if err != nil {
				return protoRound{}, err

			}

			// Get the ordering of the next node, circling back if at the last node
			nextNode := nodes[(i+1)%len(nodes)]
			nextRegion, err := getRegion(nextNode.GetOrdering())
			if err != nil {
				return protoRound{}, err

			}

			// Calculate the distance and pull the latency from the table
			totalLatency += latencyMap[thisRegion][nextRegion]

			// Stop if this permutation has already accumulated
			// a latency worse than the best time
			//  and move to next iteration
			if totalLatency > optimalLatency {
				break
			}
		}

		// Replace with the best time and order found thus far
		if totalLatency < optimalLatency {
			optimalTeam = nodes
			optimalLatency = totalLatency
		}

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

	return
}

// Creates a latency table which maps different regions latencies to all
// other defined regions. Latency is derived through educated guesses right now
// without any real world data.
// todo: table needs better real-world accuracy. Once data is collected
//  this table can be updated for better accuracy and selection
func createLatencyTable() (distanceLatency [8][8]int) {

	// Latency from Americas to other regions
	distanceLatency[Americas][Americas] = 1
	distanceLatency[Americas][WesternEurope] = 2
	distanceLatency[Americas][CentralEurope] = 4
	distanceLatency[Americas][EasternEurope] = 6
	distanceLatency[Americas][MiddleEast] = 7
	distanceLatency[Americas][Africa] = 6
	distanceLatency[Americas][Russia] = 7
	distanceLatency[Americas][Asia] = 3

	// Latency from Western Europe to other regions
	distanceLatency[WesternEurope][Americas] = 2
	distanceLatency[WesternEurope][WesternEurope] = 1
	distanceLatency[WesternEurope][CentralEurope] = 2
	distanceLatency[WesternEurope][EasternEurope] = 3
	distanceLatency[WesternEurope][MiddleEast] = 4
	distanceLatency[WesternEurope][Africa] = 2
	distanceLatency[WesternEurope][Russia] = 6
	distanceLatency[WesternEurope][Asia] = 6

	// Latency from Central Europe to other regions
	distanceLatency[CentralEurope][Americas] = 4
	distanceLatency[CentralEurope][WesternEurope] = 2
	distanceLatency[CentralEurope][CentralEurope] = 1
	distanceLatency[CentralEurope][EasternEurope] = 2
	distanceLatency[CentralEurope][MiddleEast] = 5
	distanceLatency[CentralEurope][Africa] = 2
	distanceLatency[CentralEurope][Russia] = 5
	distanceLatency[CentralEurope][Asia] = 5

	// Latency from Eastern Europe to other regions
	distanceLatency[EasternEurope][Americas] = 6
	distanceLatency[EasternEurope][WesternEurope] = 3
	distanceLatency[EasternEurope][CentralEurope] = 2
	distanceLatency[EasternEurope][EasternEurope] = 1
	distanceLatency[EasternEurope][MiddleEast] = 2
	distanceLatency[EasternEurope][Africa] = 4
	distanceLatency[EasternEurope][Russia] = 2
	distanceLatency[EasternEurope][Asia] = 4

	// Latency from Middle_East to other regions
	distanceLatency[MiddleEast][Americas] = 7
	distanceLatency[MiddleEast][WesternEurope] = 4
	distanceLatency[MiddleEast][CentralEurope] = 5
	distanceLatency[MiddleEast][EasternEurope] = 2
	distanceLatency[MiddleEast][MiddleEast] = 1
	distanceLatency[MiddleEast][Africa] = 6
	distanceLatency[MiddleEast][Russia] = 5
	distanceLatency[MiddleEast][Asia] = 2

	// Latency from Africa to other regions
	distanceLatency[Africa][Americas] = 6
	distanceLatency[Africa][WesternEurope] = 2
	distanceLatency[Africa][CentralEurope] = 2
	distanceLatency[Africa][EasternEurope] = 4
	distanceLatency[Africa][MiddleEast] = 6
	distanceLatency[Africa][Africa] = 1
	distanceLatency[Africa][Russia] = 7
	distanceLatency[Africa][Asia] = 6

	// Latency from Russia to other regions
	distanceLatency[Russia][Americas] = 7
	distanceLatency[Russia][WesternEurope] = 6
	distanceLatency[Russia][CentralEurope] = 5
	distanceLatency[Russia][EasternEurope] = 2
	distanceLatency[Russia][MiddleEast] = 5
	distanceLatency[Russia][Africa] = 7
	distanceLatency[Russia][Russia] = 1
	distanceLatency[Russia][Asia] = 2

	// Latency from Asia to other regions
	distanceLatency[Asia][Americas] = 3
	distanceLatency[Asia][WesternEurope] = 6
	distanceLatency[Asia][CentralEurope] = 5
	distanceLatency[Asia][EasternEurope] = 4
	distanceLatency[Asia][MiddleEast] = 2
	distanceLatency[Asia][Africa] = 6
	distanceLatency[Asia][Russia] = 2
	distanceLatency[Asia][Asia] = 1

	return
}

// Convert region to a numerical representation
//  in order to find distances between regions
// fixme: consider modifying node state for a region field?
func getRegion(region string) (int, error) {
	switch region {
	case "Americas":
		return Americas, nil
	case "WesternEurope":
		return WesternEurope, nil
	case "CentralEurope":
		return CentralEurope, nil
	case "EasternEurope":
		return EasternEurope, nil
	case "MiddleEast":
		return MiddleEast, nil
	case "Africa":
		return Africa, nil
	case "Russia":
		return Russia, nil
	case "Asia":
		return Asia, nil
	default:
		return -1, errors.Errorf("Could not parse region info ('%s')", region)

	}
}
