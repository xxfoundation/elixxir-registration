////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package scheduling

// Contains the scheduling params object and the internal protoRound object

import (
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/comms/connect"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

// This exists to provide thread-safe functionality to the Params object
// and to allow making safe copies of the internal Params object
type SafeParams struct {
	// Need a mutex as params can be modified out of band
	sync.RWMutex

	// Hold a reference to the actual Params
	*Params
}

// Allows for safe duplication of the current internal Params object
func (s *SafeParams) safeCopy() Params {
	s.RLock()
	defer s.RUnlock()
	return *s.Params
}

// JSONable structure which defines the parameters of the Scheduler
type Params struct {
	// selects if the secure or simple node selection algorithm is used
	Secure bool
	// number of nodes in a team
	TeamSize uint32
	// number of slots in a batch
	BatchSize uint32

	// NOTE: All times in MS
	// Resource queue timeout on nodes
	ResourceQueueTimeout time.Duration
	// Time between assigning a round
	MinimumDelay time.Duration
	// Delay for a realtime round to start
	RealtimeDelay time.Duration
	// Time between cleaning up offline nodes
	NodeCleanUpInterval time.Duration
	// Time until round precomputation times out
	PrecomputationTimeout time.Duration
	// Time until round realtime times out
	RealtimeTimeout time.Duration
	//Debug flag used to cause regular prints about the state of the network
	DebugTrackRounds bool

	//SECURE ONLY
	// sets the minimum number of nodes in the waiting pool before secure teaming
	// wil create a team
	Threshold uint32
}

//internal structure which describes a round to be created
type protoRound struct {
	Topology             *connect.Circuit
	ID                   id.Round
	NodeStateList        []*node.State
	BatchSize            uint32
	ResourceQueueTimeout time.Duration
}
