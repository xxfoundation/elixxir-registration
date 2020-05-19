////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

import "gitlab.com/elixxir/registration/storage/node"

// permute.go contains the implementation of Heap's algorithm, used to generate all
// possible permutations of n objects

// Based off of Heap's algorithm found here: https://en.wikipedia.org/wiki/Heap%27s_algorithm.
// Runs n! time, but in place in terms of space. As of writing, we use this for permuting all
// orders of a team, of which team size is small, justifying the high complexity
// TODO: consider moving this to primitives, seems there can be generic uses for this
func Permute(items []*node.State) [][]*node.State {
	var helper func([]*node.State, int)
	var output [][]*node.State

	// Place inline to make appending output easier
	helper = func(items []*node.State, n int) {
		if n == 1 {
			// Create a copy and append the copy to the output
			ourCopy := make([]*node.State, len(items))
			copy(ourCopy, items)
			output = append(output, ourCopy)
		} else {
			for i := 0; i < n; i++ {
				helper(items, n-1)
				// Swap choice dependent on parity of k (even or odd)
				if n%2 == 1 {
					// Swap the values
					items[i], items[n-1] = items[n-1], items[i]

				} else {
					// Swap the values
					items[0], items[n-1] = items[n-1], items[0]

				}
			}
		}
	}

	// Initialize recursive function
	helper(items, len(items))
	return output
}
