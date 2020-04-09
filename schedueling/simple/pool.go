package simple

import "gitlab.com/elixxir/primitives/id"

type waitingPoll struct{
	pool []*id.Node

	size int
	position int
}

func newWaitingPool(size int)*waitingPoll{
	return &waitingPoll{
		pool: make([]*id.Node)
	}
}