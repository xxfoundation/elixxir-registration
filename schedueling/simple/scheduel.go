package simple

import (
	"container/ring"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
)

func SimpleScheduler(state *storage.NetworkState)error{

	pool := newWaitingPool(state.GetNodeMap().Len())

	roundIncrement := false

	for nodeUpdate := range state.GetNodeUpdateChannel(){
		switch nodeUpdate.To{
		case current.WAITING:
			if nodeUpdate.From != current.NOT_STARTED && nodeUpdate.From != current.COMPLETED{
				return errors.New("Cannot enter ")
			}

			if is in round{
				//error
			}

			err := pool.Add(nodeUpdate.Node)
			if err!=nil{
				return errors.WithMessage(err,
					"Waiting Pool add should never error")
			}


		case current.PRECOMPUTING:
			if nodeUpdate.From != current.WAITING{
				return errors.Errorf("Node %s Cannot enter %s from " +
					"activity other than ")
			}
			n := state.GetNodeMap().GetNode(nodeUpdate.Node)
			hasRound, r := n.GetCurrentRound()
			if !hasRound{
				return errors.Errorf("Node %s should not have entered " +
					"%s without being part of a round", nodeUpdate.Node,
					current.PRECOMPUTING)
			}
			r := state.GetRoundMap().GetRound(n.)
		}




	}


}

