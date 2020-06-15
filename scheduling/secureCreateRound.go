package scheduling

import (
	"fmt"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"math"
	"time"
)

const (
	Americas      = 0
	WesternEurope = 1
	CentralEurope = 2
	EasternEurope = 3
	MiddleEast    = 4
	Africa        = 5
	Russia        = 6
	Asia          = 7
)

// createSimpleRound builds the team for a round of a pool and round id
func createSecureRound(params Params, pool *waitingPool, roundID id.Round,
	state *storage.NetworkState) (protoRound, error) {

	// Create a latencyTable (todo: have this table be based on better data)
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
	// This assumes order is geographic region, where (arbitrarily)
	// Americas       - Entirety of North and South America
	// Western Europe - todo define countries in region
	// Central Europe - todo define countries in region
	// Eastern Europe - todo define countries in region
	// Middle East    - todo define countries in region
	// Africa         - Consists of entire continent of Africa
	// Russia         - Consists of the country of Russia
	// Asia           - todo define countries in region
	// We shall assume geographical distance causes latency in a naive
	//  manner, as delineated here:
	//  https://docs.google.com/document/d/1oyjIDlqC54u_eoFzQP9SVNU2IqjnQOjpUYd9aqbg5X0/edit#

	jww.DEBUG.Printf("Looking for most efficient teaming order")
	bestTime := math.MaxInt32
	var bestOrder []*node.State
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

	jww.DEBUG.Printf("Permuting and finding the best team took: %v", time.Now().Sub(start))
	fmt.Printf("Permuting and finding the best team took: %v\n", time.Now().Sub(start))
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
func createLatencyTable() (distanceLatency [8][8]int) {

	// Distance from Americas to other regions
	distanceLatency[Americas][Americas] = 50
	distanceLatency[Americas][WesternEurope] = 200
	distanceLatency[Americas][CentralEurope] = 250
	distanceLatency[Americas][EasternEurope] = 300
	distanceLatency[Americas][MiddleEast] = 380
	distanceLatency[Americas][Africa] = 450
	distanceLatency[Americas][Russia] = 500
	distanceLatency[Americas][Asia] = 550

	// Distance from Western Europe to other regions
	distanceLatency[WesternEurope][Americas] = 200
	distanceLatency[WesternEurope][WesternEurope] = 50
	distanceLatency[WesternEurope][CentralEurope] = 100
	distanceLatency[WesternEurope][EasternEurope] = 150
	distanceLatency[WesternEurope][MiddleEast] = 220
	distanceLatency[WesternEurope][Africa] = 250
	distanceLatency[WesternEurope][Russia] = 200
	distanceLatency[WesternEurope][Asia] = 400

	// Distance from Central Europe to other regions
	distanceLatency[CentralEurope][Americas] = 250
	distanceLatency[CentralEurope][WesternEurope] = 100
	distanceLatency[CentralEurope][CentralEurope] = 50
	distanceLatency[CentralEurope][EasternEurope] = 100
	distanceLatency[CentralEurope][MiddleEast] = 200
	distanceLatency[CentralEurope][Africa] = 250
	distanceLatency[CentralEurope][Russia] = 200
	distanceLatency[CentralEurope][Asia] = 350

	// Distance from Eastern Europe to other regions
	distanceLatency[EasternEurope][Americas] = 300
	distanceLatency[EasternEurope][WesternEurope] = 150
	distanceLatency[EasternEurope][CentralEurope] = 100
	distanceLatency[EasternEurope][EasternEurope] = 50
	distanceLatency[EasternEurope][MiddleEast] = 150
	distanceLatency[EasternEurope][Africa] = 220
	distanceLatency[EasternEurope][Russia] = 170
	distanceLatency[EasternEurope][Asia] = 300

	// Distance from Middle_East to other regions
	distanceLatency[MiddleEast][Americas] = 380
	distanceLatency[MiddleEast][WesternEurope] = 220
	distanceLatency[MiddleEast][CentralEurope] = 200
	distanceLatency[MiddleEast][EasternEurope] = 150
	distanceLatency[MiddleEast][MiddleEast] = 50
	distanceLatency[MiddleEast][Africa] = 150
	distanceLatency[MiddleEast][Russia] = 100
	distanceLatency[MiddleEast][Asia] = 150

	// Distance from Africa to other regions
	distanceLatency[Africa][Americas] = 450
	distanceLatency[Africa][WesternEurope] = 250
	distanceLatency[Africa][CentralEurope] = 250
	distanceLatency[Africa][EasternEurope] = 220
	distanceLatency[Africa][MiddleEast] = 150
	distanceLatency[Africa][Africa] = 50
	distanceLatency[Africa][Russia] = 200
	distanceLatency[Africa][Asia] = 230

	// Distance from Russia to other regions
	distanceLatency[Russia][Americas] = 500
	distanceLatency[Russia][WesternEurope] = 200
	distanceLatency[Russia][CentralEurope] = 200
	distanceLatency[Russia][EasternEurope] = 170
	distanceLatency[Russia][MiddleEast] = 100
	distanceLatency[Russia][Africa] = 200
	distanceLatency[Russia][Russia] = 50
	distanceLatency[Russia][Asia] = 200

	// Distance from Asia to other regions
	distanceLatency[Asia][Americas] = 550
	distanceLatency[Asia][WesternEurope] = 400
	distanceLatency[Asia][CentralEurope] = 350
	distanceLatency[Asia][EasternEurope] = 300
	distanceLatency[Asia][MiddleEast] = 150
	distanceLatency[Asia][Africa] = 230
	distanceLatency[Asia][Russia] = 200
	distanceLatency[Asia][Asia] = 50

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
