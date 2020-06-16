package scheduling

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage/node"
	"math"
)

// Create a latencyTable
var latencyTable = createLatencyTable()

func generateSemiOptimalOrdering(nodes []*node.State)([]*node.State,error){
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
				return nil, err
			}

			// Get the ordering of the next node, circling back if at the last node
			nextNode := nodes[(i+1)%len(nodes)]
			nextRegion, err := getRegion(nextNode.GetOrdering())
			if err != nil {
				return nil, err

			}

			// Calculate the distance and pull the latency from the table
			totalLatency += latencyTable[thisRegion][nextRegion]

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
	return optimalTeam, nil
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
	distanceLatency[Americas][MiddleEast] = 13
	distanceLatency[Americas][Africa] = 6
	distanceLatency[Americas][Russia] = 4
	distanceLatency[Americas][Asia] = 2 // america -> central euro -> rus. (13) or america -> asia russia (4)

	// Latency from Western Europe to other regions
	distanceLatency[WesternEurope][Americas] = 2
	distanceLatency[WesternEurope][WesternEurope] = 1
	distanceLatency[WesternEurope][CentralEurope] = 2
	distanceLatency[WesternEurope][EasternEurope] = 4
	distanceLatency[WesternEurope][MiddleEast] = 6
	distanceLatency[WesternEurope][Africa] = 2
	distanceLatency[WesternEurope][Russia] = 6 // w euro -> e. euro -> rus.
	distanceLatency[WesternEurope][Asia] = 13  // w. euro -> c. euro -> mid east -> asia (13)

	// Latency from Central Europe to other regions
	distanceLatency[CentralEurope][Americas] = 4
	distanceLatency[CentralEurope][WesternEurope] = 2
	distanceLatency[CentralEurope][CentralEurope] = 1
	distanceLatency[CentralEurope][EasternEurope] = 2
	distanceLatency[CentralEurope][MiddleEast] = 2
	distanceLatency[CentralEurope][Africa] = 2
	distanceLatency[CentralEurope][Russia] = 4
	distanceLatency[CentralEurope][Asia] = 6 //

	// Latency from Eastern Europe to other regions
	distanceLatency[EasternEurope][Americas] = 6
	distanceLatency[EasternEurope][WesternEurope] = 4
	distanceLatency[EasternEurope][CentralEurope] = 2
	distanceLatency[EasternEurope][EasternEurope] = 1
	distanceLatency[EasternEurope][MiddleEast] = 2
	distanceLatency[EasternEurope][Africa] = 4
	distanceLatency[EasternEurope][Russia] = 2
	distanceLatency[EasternEurope][Asia] = 4

	// Latency from Middle_East to other regions
	distanceLatency[MiddleEast][Americas] = 13
	distanceLatency[MiddleEast][WesternEurope] = 4
	distanceLatency[MiddleEast][CentralEurope] = 2
	distanceLatency[MiddleEast][EasternEurope] = 2
	distanceLatency[MiddleEast][MiddleEast] = 1
	distanceLatency[MiddleEast][Africa] = 4 // c. euro to africe (4) or e. euro to c. euro to africa (6)?
	distanceLatency[MiddleEast][Russia] = 4
	distanceLatency[MiddleEast][Asia] = 2

	// Latency from Africa to other regions
	distanceLatency[Africa][Americas] = 6
	distanceLatency[Africa][WesternEurope] = 2
	distanceLatency[Africa][CentralEurope] = 2
	distanceLatency[Africa][EasternEurope] = 4
	distanceLatency[Africa][MiddleEast] = 4
	distanceLatency[Africa][Africa] = 1
	distanceLatency[Africa][Russia] = 6
	distanceLatency[Africa][Asia] = 6 // c. euro to mid east to asia

	// Latency from Russia to other regions
	distanceLatency[Russia][Americas] = 4
	distanceLatency[Russia][WesternEurope] = 13
	distanceLatency[Russia][CentralEurope] = 6
	distanceLatency[Russia][EasternEurope] = 4
	distanceLatency[Russia][MiddleEast] = 4
	distanceLatency[Russia][Africa] = 6
	distanceLatency[Russia][Russia] = 1
	distanceLatency[Russia][Asia] = 2

	// Latency from Asia to other regions
	distanceLatency[Asia][Americas] = 3
	distanceLatency[Asia][WesternEurope] = 6
	distanceLatency[Asia][CentralEurope] = 6
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
