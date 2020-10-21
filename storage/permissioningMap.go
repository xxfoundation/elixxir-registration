////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the MapImpl for permissioning-based functionality

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
)

// Inserts the given State into database if it does not exist
// Or updates the database State if its value does not match the given State
func (m *MapImpl) UpsertState(state *State) error {
	// TODO & Test
	return nil
}

// Returns a State's value from database with the given key
// Or an error if a matching State does not exist
func (m *MapImpl) GetStateValue(key string) (string, error) {
	// TODO & Test
	return "", nil
}

// Insert NodeMetric object
func (m *MapImpl) InsertNodeMetric(metric *NodeMetric) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	// Auto-increment key
	m.nodeMetricCounter += 1

	// Add to map
	metric.Id = m.nodeMetricCounter
	jww.DEBUG.Printf("Attempting to insert node metric: %+v", metric)
	m.nodeMetrics[m.nodeMetricCounter] = metric
	return nil
}

// Insert RoundError object
func (m *MapImpl) InsertRoundError(roundId id.Round, errStr string) error {
	m.mut.Lock()
	defer m.mut.Unlock()
	rid := uint64(roundId)

	m.roundMetrics[rid].RoundErrors = append(
		m.roundMetrics[rid].RoundErrors,
		RoundError{
			Id:            0, // Currently useless in MapImpl
			RoundMetricId: rid,
			Error:         errStr,
		},
	)
	return nil
}

// Insert RoundMetric object with associated topology
func (m *MapImpl) InsertRoundMetric(metric *RoundMetric, topology [][]byte) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	// Build Topology objects
	metric.Topologies = make([]Topology, len(topology))
	for i, nodeIdBytes := range topology {
		nodeId, err := id.Unmarshal(nodeIdBytes)
		if err != nil {
			return errors.New(err.Error())
		}
		topologyObj := Topology{
			NodeId:        nodeId.Bytes(),
			RoundMetricId: metric.Id,
			Order:         uint8(i),
		}
		metric.Topologies[i] = topologyObj
	}

	// Add to map
	jww.DEBUG.Printf("Attempting to insert round metric: %+v", metric)
	m.roundMetrics[metric.Id] = metric
	return nil
}
