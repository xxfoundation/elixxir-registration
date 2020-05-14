////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/id"
)

// pool.go contains the logic for a waiting pool, a simple list of nodes

// contains a list of nodes of a certain size. does not allow more to be added
// than its max size and all nodes must be removed at once.
type waitingPoll struct {
	pool []*id.ID

	position int
}

//creates an empty waiting of object of the designated size
func newWaitingPool(size int) *waitingPoll {
	return &waitingPoll{
		pool:     make([]*id.ID, size),
		position: 0,
	}
}

// adds an element to the waiting pool if it is not full, otherwise returns an
// error
func (wp *waitingPoll) Add(nid *id.ID) error {
	if wp.position == len(wp.pool) {
		return errors.New("waiting pool is full")
	}

	wp.pool[wp.position] = nid
	wp.position += 1

	return nil
}

// returns all elements from the waiting pool and clears it
func (wp *waitingPoll) Clear() []*id.ID {
	old := wp.pool
	wp.pool = make([]*id.ID, len(wp.pool))
	wp.position = 0
	return old
}

// returns the number of elements currently in the pool
func (wp *waitingPoll) Len() int {
	return wp.position
}

// returns the maximum size of the waiting pool
func (wp *waitingPoll) Size() int {
	return len(wp.pool)
}
