////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/comms/connect"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"math"
	"sync"
	"testing"
	"time"
)

// Tracks the current global state of a round
type State struct {
	// round info to be used to produce new round infos
	base *pb.RoundInfo

	//topology of the round
	topology *connect.Circuit

	//state of the round
	state states.Round

	// Number of nodes ready for the next transition
	readyForTransition uint8

	mux sync.RWMutex
}

//creates a round state object
func newState(id id.Round, batchsize uint32, topology *connect.Circuit, pendingTs time.Time) *State {
	strTopology := make([]string, topology.Len())
	for i := 0; i < topology.Len(); i++ {
		strTopology[i] = topology.GetNodeAtIndex(i).String()
	}

	//create the timestamps and populate the first one
	timestamps := make([]uint64, states.NUM_STATES)
	timestamps[states.PENDING] = uint64(pendingTs.Unix())

	//build and return the round state object
	return &State{
		base: &pb.RoundInfo{
			ID:         uint64(id),
			UpdateID:   math.MaxUint64,
			State:      0,
			BatchSize:  batchsize,
			Topology:   strTopology,
			Timestamps: timestamps,
		},
		topology:           topology,
		state:              states.PENDING,
		readyForTransition: 0,
		mux:                sync.RWMutex{},
	}
}

//creates a round state object
func NewState_Testing(id id.Round, state states.Round, t *testing.T) *State {
	if t == nil {
		jww.FATAL.Panic("Only for testing")
	}
	//build and return the round state object
	return &State{
		base: &pb.RoundInfo{
			ID:         uint64(id),
			Timestamps: make([]uint64, states.NUM_STATES),
		},
		state:              state,
		readyForTransition: 0,
		mux:                sync.RWMutex{},
	}
}

// Increments that another node is ready for a transition.
// and returns true and clears if the transition is ready
func (s *State) NodeIsReadyForTransition() bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.readyForTransition++
	if int(s.readyForTransition) == s.topology.Len() {
		s.readyForTransition = 0
		return true
	}
	return false
}

// updates the round to a new state. states can only move forward, they cannot
// go in reverse or replace the same state
func (s *State) Update(state states.Round, stamp time.Time) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	if state <= s.state {
		return errors.New("round state must always update to a " +
			"greater state")
	}

	s.state = state
	s.base.Timestamps[state] = uint64(stamp.Unix())
	return nil
}

//returns an unsigned roundinfo with all fields filled in
func (s *State) BuildRoundInfo() *pb.RoundInfo {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return &pb.RoundInfo{
		ID:         s.base.GetID(),
		State:      uint32(s.state),
		BatchSize:  s.base.GetBatchSize(),
		Topology:   s.base.GetTopology(),
		Timestamps: s.base.GetTimestamps(),
	}
}

//returns the state of the round
func (s *State) GetRoundState() states.Round {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.state
}

//returns the round's topology
func (s *State) GetTopology() *connect.Circuit {
	return s.topology
}

//returns the id of the round
func (s *State) GetRoundID() id.Round {
	rid := id.Round(s.base.ID)
	return rid
}
