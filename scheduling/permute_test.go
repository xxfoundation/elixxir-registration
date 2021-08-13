////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import (
	"fmt"
	"git.xx.network/elixxir/registration/storage/node"
	"strconv"
	"testing"
)

// Happy path
func TestPermute(t *testing.T) {

	testState := setupNodeMap(t)

	totalNodes := 3
	nodeList := make([]*node.State, totalNodes)

	// Build node states with unique ordering
	for i := 0; i < totalNodes; i++ {
		// Make a node state
		newNode := setupNode(t, testState, uint64(i))

		// set the order of the
		newOrder := strconv.Itoa(i)
		newNode.SetOrdering(newOrder)

		// Place new node in list
		nodeList[i] = newNode

	}

	// Permute the nodes
	permutations := Permute(nodeList)
	expectedLen := factorial(totalNodes)

	// Verify that the amount of permutations is
	// factorial of the original amount of nodes
	if len(permutations) != expectedLen {
		t.Errorf("Permutations did not produce the expected amount of permutations "+
			"(factorial of amount of nodes)!"+
			"\n\tExpected: %d"+
			"\n\tReceived: %d", expectedLen, len(permutations))
	}

	expectedPermutations := make(map[string]bool)

	// Iterate through all the permutations to ensure uniqueness between orderings
	for _, permutation := range permutations {
		var concatenatedOrdering string
		// Concatenate orderings into a single string
		for _, ourNode := range permutation {
			concatenatedOrdering += ourNode.GetOrdering()
		}
		// If that ordering has been encountered before, error
		if expectedPermutations[concatenatedOrdering] {
			t.Errorf("Permutation %s has occurred more than once!", concatenatedOrdering)
		}

		// Mark permutation as seen
		expectedPermutations[concatenatedOrdering] = true

	}

}

func factorial(n int) int {
	factVal := 1
	if n < 0 {
		fmt.Println("Factorial of negative number doesn't exist.")
	} else {
		for i := 1; i <= n; i++ {
			factVal *= i
		}

	}
	return factVal
}
