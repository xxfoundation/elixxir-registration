package simple

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/id"
)

type waitingPoll struct {
	pool []*id.Node

	position int
}

//creates an empty waiting of object of the designated size
func newWaitingPool(size int) *waitingPoll {
	return &waitingPoll{
		pool:     make([]*id.Node, size),
		position: 0,
	}
}

// adds an element to the waiting pool if it is not full, otherwise returns an
// error
func (wp *waitingPoll) Add(nid *id.Node) error {
	if wp.position+1 == len(wp.pool) {
		return errors.New("waiting pool is full")
	}

	wp.position += 1
	wp.pool[wp.position] = nid
	return nil
}

// returns all elements from the waiting pool and clears it
func (wp *waitingPoll) Clear() []*id.Node {
	old := wp.pool
	wp.pool = make([]*id.Node, len(wp.pool))
	wp.position = 0
	return old
}

// returns the number of elements currently in the pool
func (wp *waitingPoll) Len() int {
	return wp.position + 1
}

// returns the maximum size of the waiting pool
func (wp *waitingPoll) Size() int {
	return len(wp.pool)
}
