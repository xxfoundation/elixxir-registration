package simple

import (
	"container/ring"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/registration/storage"
)

func SimpleScheduler(state *storage.NetworkState)error{

	waitingPool := ring.New()


	for nodeUpdate := range state.GetNodeUpdateChannel(){





	}


}

