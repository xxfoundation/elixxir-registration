package simple

import (
	"encoding/json"
	"github.com/pkg/errors"
	"gitlab.com/elixxir/registration/storage"
)

type Params struct{
	TeamSize uint32
	BatchSize uint32
	RandomOrdering bool
}

func Scheduler(serialParam []byte, state *storage.NetworkState)error{
	var params Params
	err := json.Unmarshal(serialParam, &params)
	if err!=nil{
		return errors.WithMessage(err, "Could not extract parameters")
	}

	return scheduler(params, state)
}

func scheduler(params Params, state *storage.NetworkState)error{

	pool := newWaitingPool(state.GetNodeMap().Len())

	roundID := NewRoundID(0)
	updateID := NewUpdateID(0)

	for update := range state.GetNodeUpdateChannel(){

		//handle the node's state change
		err := HandleNodeStateChance(update, pool, updateID, state)
		if err!=nil{
			return err
		}

		//create a new round if the pool is full
		if pool.Len()==int(params.TeamSize) {
			err = createRound(params, pool, roundID, updateID, state)
			if err!=nil{
				return err
			}
		}

	}

	return errors.New("single scheduler should never exit")
}


