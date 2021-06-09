package scheduling

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/geobins"
	"math"
)

// Create a latencyTable
var latencyTable = createLatencyTable()

func generateSemiOptimalOrdering(nodes []*node.State) ([]*node.State, error) {
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
			thisRegion, err := geobins.GetRegion(nodes[i].GetOrdering())
			if err != nil {
				return nil, err
			}

			// Get the ordering of the next node, circling back if at the last node
			nextNode := nodes[(i+1)%len(nodes)]
			nextRegion, err := geobins.GetRegion(nextNode.GetOrdering())
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

	// Latency from geobins.Americas to other regions
	distanceLatency[geobins.Americas][geobins.Americas] = 1
	distanceLatency[geobins.Americas][geobins.WesternEurope] = 2
	distanceLatency[geobins.Americas][geobins.CentralEurope] = 4
	distanceLatency[geobins.Americas][geobins.EasternEurope] = 6
	distanceLatency[geobins.Americas][geobins.MiddleEast] = 13
	distanceLatency[geobins.Americas][geobins.Africa] = 6
	distanceLatency[geobins.Americas][geobins.Russia] = 4
	distanceLatency[geobins.Americas][geobins.Asia] = 2 // america -> central euro -> rus. (13) or america -> geobins.Asia geobins.Russia (4)

	// Latency from Western Europe to other regions
	distanceLatency[geobins.WesternEurope][geobins.Americas] = 2
	distanceLatency[geobins.WesternEurope][geobins.WesternEurope] = 1
	distanceLatency[geobins.WesternEurope][geobins.CentralEurope] = 2
	distanceLatency[geobins.WesternEurope][geobins.EasternEurope] = 4
	distanceLatency[geobins.WesternEurope][geobins.MiddleEast] = 6
	distanceLatency[geobins.WesternEurope][geobins.Africa] = 2
	distanceLatency[geobins.WesternEurope][geobins.Russia] = 6 // w euro -> e. euro -> rus.
	distanceLatency[geobins.WesternEurope][geobins.Asia] = 13  // w. euro -> c. euro -> mid east -> geobins.Asia (13)

	// Latency from Central Europe to other regions
	distanceLatency[geobins.CentralEurope][geobins.Americas] = 4
	distanceLatency[geobins.CentralEurope][geobins.WesternEurope] = 2
	distanceLatency[geobins.CentralEurope][geobins.CentralEurope] = 1
	distanceLatency[geobins.CentralEurope][geobins.EasternEurope] = 2
	distanceLatency[geobins.CentralEurope][geobins.MiddleEast] = 2
	distanceLatency[geobins.CentralEurope][geobins.Africa] = 2
	distanceLatency[geobins.CentralEurope][geobins.Russia] = 4
	distanceLatency[geobins.CentralEurope][geobins.Asia] = 6 //

	// Latency from Eastern Europe to other regions
	distanceLatency[geobins.EasternEurope][geobins.Americas] = 6
	distanceLatency[geobins.EasternEurope][geobins.WesternEurope] = 4
	distanceLatency[geobins.EasternEurope][geobins.CentralEurope] = 2
	distanceLatency[geobins.EasternEurope][geobins.EasternEurope] = 1
	distanceLatency[geobins.EasternEurope][geobins.MiddleEast] = 2
	distanceLatency[geobins.EasternEurope][geobins.Africa] = 4
	distanceLatency[geobins.EasternEurope][geobins.Russia] = 2
	distanceLatency[geobins.EasternEurope][geobins.Asia] = 4

	// Latency from Middle_East to other regions
	distanceLatency[geobins.MiddleEast][geobins.Americas] = 13
	distanceLatency[geobins.MiddleEast][geobins.WesternEurope] = 4
	distanceLatency[geobins.MiddleEast][geobins.CentralEurope] = 2
	distanceLatency[geobins.MiddleEast][geobins.EasternEurope] = 2
	distanceLatency[geobins.MiddleEast][geobins.MiddleEast] = 1
	distanceLatency[geobins.MiddleEast][geobins.Africa] = 4 // c. euro to africe (4) or e. euro to c. euro to geobins.Africa (6)?
	distanceLatency[geobins.MiddleEast][geobins.Russia] = 4
	distanceLatency[geobins.MiddleEast][geobins.Asia] = 2

	// Latency from geobins.Africa to other regions
	distanceLatency[geobins.Africa][geobins.Americas] = 6
	distanceLatency[geobins.Africa][geobins.WesternEurope] = 2
	distanceLatency[geobins.Africa][geobins.CentralEurope] = 2
	distanceLatency[geobins.Africa][geobins.EasternEurope] = 4
	distanceLatency[geobins.Africa][geobins.MiddleEast] = 4
	distanceLatency[geobins.Africa][geobins.Africa] = 1
	distanceLatency[geobins.Africa][geobins.Russia] = 6
	distanceLatency[geobins.Africa][geobins.Asia] = 6 // c. euro to mid east to geobins.Asia

	// Latency from geobins.Russia to other regions
	distanceLatency[geobins.Russia][geobins.Americas] = 4
	distanceLatency[geobins.Russia][geobins.WesternEurope] = 13
	distanceLatency[geobins.Russia][geobins.CentralEurope] = 6
	distanceLatency[geobins.Russia][geobins.EasternEurope] = 4
	distanceLatency[geobins.Russia][geobins.MiddleEast] = 4
	distanceLatency[geobins.Russia][geobins.Africa] = 6
	distanceLatency[geobins.Russia][geobins.Russia] = 1
	distanceLatency[geobins.Russia][geobins.Asia] = 2

	// Latency from geobins.Asia to other regions
	distanceLatency[geobins.Asia][geobins.Americas] = 3
	distanceLatency[geobins.Asia][geobins.WesternEurope] = 6
	distanceLatency[geobins.Asia][geobins.CentralEurope] = 6
	distanceLatency[geobins.Asia][geobins.EasternEurope] = 4
	distanceLatency[geobins.Asia][geobins.MiddleEast] = 2
	distanceLatency[geobins.Asia][geobins.Africa] = 6
	distanceLatency[geobins.Asia][geobins.Russia] = 2
	distanceLatency[geobins.Asia][geobins.Asia] = 1

	return
}
