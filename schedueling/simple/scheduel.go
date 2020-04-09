package simple

import (
	"container/ring"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
)

func SimpleScheduler(state *storage.NetworkState)error{

	waitingPool := make([]*id.Node, state.GetNodeMap().Len())
	position

	waitingPool.

	for nodeUpdate := range state.GetNodeUpdateChannel(){





	}


}

