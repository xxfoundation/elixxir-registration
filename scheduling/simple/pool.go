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

type waitingPoll struct {
	pool []*id.Node

	position int
}

const emptyPool = 0

//creates an empty waiting of object of the designated size
func newWaitingPool(size int) *waitingPoll {
	return &waitingPoll{
		pool:     make([]*id.Node, size),
		position: emptyPool,
	}
}

// adds an element to the waiting pool if it is not full, otherwise returns an
// error
func (wp *waitingPoll) Add(nid *id.Node) error {
	if wp.position == len(wp.pool) {
		return errors.New("waiting pool is full")
	}

	wp.pool[wp.position] = nid
	wp.position += 1

	return nil
}

// returns all elements from the waiting pool and clears it
func (wp *waitingPoll) Clear() []*id.Node {
	old := wp.pool
	wp.pool = make([]*id.Node, len(wp.pool))
	wp.position = emptyPool
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
