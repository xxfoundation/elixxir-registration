////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package round

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
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

	// List of round errors received from nodes
	roundErrors []*pb.RoundError

	// List of client errors received from nodes
	clientErrors []*pb.ClientError

	roundComplete chan struct{}

	lastUpdate time.Time

	// Keep track of the ns timestamp when the last node in the round reported completed
	// in order to get better granularity for when realtime finished
	realtimeCompletedTs int64

	mux sync.RWMutex
}

// creates a round state object
func newState(id id.Round, batchsize, addressSpaceSize uint32, resourceQueueTimeout time.Duration,
	topology *connect.Circuit, pendingTs time.Time) *State {

	strTopology := make([][]byte, topology.Len())
	for i := 0; i < topology.Len(); i++ {
		strTopology[i] = topology.GetNodeAtIndex(i).Marshal()
	}

	//create the timestamps and populate the first one
	timestamps := make([]uint64, states.NUM_STATES)
	timestamps[states.PENDING] = uint64(pendingTs.Unix())

	roundCompleteChan := make(chan struct{}, 1)

	//build and return the round state object
	return &State{
		base: &pb.RoundInfo{
			ID:                         uint64(id),
			UpdateID:                   math.MaxUint64,
			State:                      0,
			BatchSize:                  batchsize,
			Topology:                   strTopology,
			Timestamps:                 timestamps,
			ResourceQueueTimeoutMillis: uint32(resourceQueueTimeout),
			AddressSpaceSize:           addressSpaceSize,
		},
		topology:           topology,
		state:              states.PENDING,
		readyForTransition: 0,
		mux:                sync.RWMutex{},
		roundComplete:      roundCompleteChan,
	}
}

// creates a round state object
func NewState_Testing(id id.Round, state states.Round, topology *connect.Circuit, t *testing.T) *State {
	if t == nil {
		jww.FATAL.Panic("Only for testing")
	}

	roundCompleteChan := make(chan struct{}, 1000)

	//build and return the round state object
	return &State{
		base: &pb.RoundInfo{
			ID:         uint64(id),
			Timestamps: make([]uint64, states.NUM_STATES),
		},
		state:              state,
		readyForTransition: 0,
		mux:                sync.RWMutex{},
		roundComplete:      roundCompleteChan,
		topology:           topology,
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

	s.lastUpdate = time.Now()

	s.state = state
	s.base.Timestamps[state] = uint64(stamp.UnixNano())
	return nil
}

// returns the last time the round was updated
func (s *State) GetLastUpdate() time.Time {
	s.mux.Lock()
	defer s.mux.Unlock()

	return s.lastUpdate
}

// returns an unsigned roundinfo with all fields filled in
// everything must be deep copied to ensure future edits do not edit the output
// and cause signature verification failures
func (s *State) BuildRoundInfo() *pb.RoundInfo {
	s.mux.RLock()
	defer s.mux.RUnlock()

	s.base.Errors = s.roundErrors
	s.base.ClientErrors = s.clientErrors
	s.base.State = uint32(s.state)

	return CopyRoundInfo(s.base)
}

// returns the state of the round
func (s *State) GetRoundState() states.Round {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return s.state
}

// returns the round's topology
func (s *State) GetTopology() *connect.Circuit {
	return s.topology
}

// returns the id of the round
func (s *State) GetRoundID() id.Round {
	rid := id.Round(s.base.ID)
	return rid
}

// Return firstCompletedTs
func (s *State) GetRealtimeCompletedTs() int64 {
	return s.realtimeCompletedTs
}

// Set firstCompletedTs
func (s *State) SetRealtimeCompletedTs(ts int64) {
	s.realtimeCompletedTs = ts
}

// Append a round error to our list of stored rounderrors
func (s *State) AppendError(roundError *pb.RoundError) {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, e := range s.roundErrors {
		if e.String() == roundError.String() {
			return
		}
	}

	s.roundErrors = append(s.roundErrors, roundError)
}

// Append a round error to our list of stored rounderrors
func (s *State) AppendClientErrors(clientErrors []*pb.ClientError) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.clientErrors = append(s.clientErrors, clientErrors...)
}

// returns the channel used to stop the round timeout
func (s *State) GetRoundCompletedChan() <-chan struct{} {
	return s.roundComplete
}

// DenoteRoundCompleted sends on the round complete channel to the
// timeout thread to notify it that the round has completed
func (s *State) DenoteRoundCompleted() {
	select {
	case s.roundComplete <- struct{}{}:
	default:
	}
}
