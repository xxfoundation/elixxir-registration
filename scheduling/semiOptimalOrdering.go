package scheduling

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/region"
	"math"
)

// Create a latencyTable
var latencyTable = createLatencyTable()

func generateSemiOptimalOrdering(nodes []*node.State, state *storage.NetworkState) ([]*node.State, error) {
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
			thisRegion, ok := state.GetGeoBins()[nodes[i].GetOrdering()]
			if !ok {
				return nil, errors.Errorf("Unable to locate bin for code %s", nodes[i].GetOrdering())
			}

			// Get the ordering of the next node, circling back if at the last node
			nextNode := nodes[(i+1)%len(nodes)]
			nextRegion, ok := state.GetGeoBins()[nextNode.GetOrdering()]
			if !ok {
				return nil, errors.Errorf("Unable to locate bin for code %s", nextNode.GetOrdering())
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

	// Latency from region.Americas to other regions
	distanceLatency[region.Americas][region.Americas] = 1
	distanceLatency[region.Americas][region.WesternEurope] = 2
	distanceLatency[region.Americas][region.CentralEurope] = 4
	distanceLatency[region.Americas][region.EasternEurope] = 6
	distanceLatency[region.Americas][region.MiddleEast] = 13
	distanceLatency[region.Americas][region.Africa] = 6
	distanceLatency[region.Americas][region.Russia] = 4
	distanceLatency[region.Americas][region.Asia] = 2 // america -> central euro -> rus. (13) or america -> region.Asia region.Russia (4)

	// Latency from Western Europe to other regions
	distanceLatency[region.WesternEurope][region.Americas] = 2
	distanceLatency[region.WesternEurope][region.WesternEurope] = 1
	distanceLatency[region.WesternEurope][region.CentralEurope] = 2
	distanceLatency[region.WesternEurope][region.EasternEurope] = 4
	distanceLatency[region.WesternEurope][region.MiddleEast] = 6
	distanceLatency[region.WesternEurope][region.Africa] = 2
	distanceLatency[region.WesternEurope][region.Russia] = 6 // w euro -> e. euro -> rus.
	distanceLatency[region.WesternEurope][region.Asia] = 13  // w. euro -> c. euro -> mid east -> region.Asia (13)

	// Latency from Central Europe to other regions
	distanceLatency[region.CentralEurope][region.Americas] = 4
	distanceLatency[region.CentralEurope][region.WesternEurope] = 2
	distanceLatency[region.CentralEurope][region.CentralEurope] = 1
	distanceLatency[region.CentralEurope][region.EasternEurope] = 2
	distanceLatency[region.CentralEurope][region.MiddleEast] = 2
	distanceLatency[region.CentralEurope][region.Africa] = 2
	distanceLatency[region.CentralEurope][region.Russia] = 4
	distanceLatency[region.CentralEurope][region.Asia] = 6 //

	// Latency from Eastern Europe to other regions
	distanceLatency[region.EasternEurope][region.Americas] = 6
	distanceLatency[region.EasternEurope][region.WesternEurope] = 4
	distanceLatency[region.EasternEurope][region.CentralEurope] = 2
	distanceLatency[region.EasternEurope][region.EasternEurope] = 1
	distanceLatency[region.EasternEurope][region.MiddleEast] = 2
	distanceLatency[region.EasternEurope][region.Africa] = 4
	distanceLatency[region.EasternEurope][region.Russia] = 2
	distanceLatency[region.EasternEurope][region.Asia] = 4

	// Latency from Middle_East to other regions
	distanceLatency[region.MiddleEast][region.Americas] = 13
	distanceLatency[region.MiddleEast][region.WesternEurope] = 4
	distanceLatency[region.MiddleEast][region.CentralEurope] = 2
	distanceLatency[region.MiddleEast][region.EasternEurope] = 2
	distanceLatency[region.MiddleEast][region.MiddleEast] = 1
	distanceLatency[region.MiddleEast][region.Africa] = 4 // c. euro to africe (4) or e. euro to c. euro to region.Africa (6)?
	distanceLatency[region.MiddleEast][region.Russia] = 4
	distanceLatency[region.MiddleEast][region.Asia] = 2

	// Latency from region.Africa to other regions
	distanceLatency[region.Africa][region.Americas] = 6
	distanceLatency[region.Africa][region.WesternEurope] = 2
	distanceLatency[region.Africa][region.CentralEurope] = 2
	distanceLatency[region.Africa][region.EasternEurope] = 4
	distanceLatency[region.Africa][region.MiddleEast] = 4
	distanceLatency[region.Africa][region.Africa] = 1
	distanceLatency[region.Africa][region.Russia] = 6
	distanceLatency[region.Africa][region.Asia] = 6 // c. euro to mid east to region.Asia

	// Latency from region.Russia to other regions
	distanceLatency[region.Russia][region.Americas] = 4
	distanceLatency[region.Russia][region.WesternEurope] = 13
	distanceLatency[region.Russia][region.CentralEurope] = 6
	distanceLatency[region.Russia][region.EasternEurope] = 4
	distanceLatency[region.Russia][region.MiddleEast] = 4
	distanceLatency[region.Russia][region.Africa] = 6
	distanceLatency[region.Russia][region.Russia] = 1
	distanceLatency[region.Russia][region.Asia] = 2

	// Latency from region.Asia to other regions
	distanceLatency[region.Asia][region.Americas] = 3
	distanceLatency[region.Asia][region.WesternEurope] = 6
	distanceLatency[region.Asia][region.CentralEurope] = 6
	distanceLatency[region.Asia][region.EasternEurope] = 4
	distanceLatency[region.Asia][region.MiddleEast] = 2
	distanceLatency[region.Asia][region.Africa] = 6
	distanceLatency[region.Asia][region.Russia] = 2
	distanceLatency[region.Asia][region.Asia] = 1

	return
}
