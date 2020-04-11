package simple

import (
	"github.com/pkg/errors"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"time"
)

func HandleNodeStateChance(update *storage.NodeUpdateNotification, pool *waitingPoll,
	updateID *UpdateID, state *storage.NetworkState)error{
	//get node and round information
	n := state.GetNodeMap().GetNode(update.Node)
	hasRound, r := n.GetCurrentRound()

	switch update.To{
	case current.NOT_STARTED:
		//do nothing
	case current.WAITING:

		// clear the round if node has one (it should unless it is
		// comming from NOT_STARTED
		if hasRound{
			n.ClearRound()
		}

		err := pool.Add(update.Node)
		if err!=nil{
			return errors.WithMessage(err,"Waiting pool should never fill")
		}
	case current.PRECOMPUTING:
		//do nothing
	case current.STANDBY:

		if !hasRound{
			return errors.Errorf("Node %s without round should " +
				"not be in %s state", update.Node, states.PRECOMPUTING)
		}

		stateComplete:=r.NodeIsReadyForTransition()
		if stateComplete{
			err := r.Update(states.REALTIME, time.Now().Add(2500*time.Millisecond))
			if err!=nil{
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.PRECOMPUTING, states.REALTIME)
			}
			err = state.AddRoundUpdate(updateID.Next(), r.BuildRoundInfo())
			if err!=nil{
				return errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.PRECOMPUTING, states.REALTIME)
			}
		}
	case current.REALTIME:
		//do nothing
	case current.COMPLETED:

		if !hasRound{
			return errors.Errorf("Node %s without round should " +
				"not be in %s state", update.Node, states.COMPLETED)
		}

		stateComplete:=r.NodeIsReadyForTransition()
		if stateComplete{
			err := r.Update(states.COMPLETED, time.Now())
			if err!=nil{
				return errors.WithMessagef(err,
					"Could not move round %v from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}
			err = state.AddRoundUpdate(updateID.Next(), r.BuildRoundInfo())
			if err!=nil{
				return errors.WithMessagef(err, "Could not issue "+
					"update for round %v transitioning from %s to %s",
					r.GetRoundID(), states.REALTIME, states.COMPLETED)
			}
		}
	}

	return nil
}

