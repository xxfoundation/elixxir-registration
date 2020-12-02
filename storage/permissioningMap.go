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

// Inserts the given State into Database if it does not exist
// Or updates the Database State if its value does not match the given State
func (m *MapImpl) UpsertState(state *State) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	m.states[state.Key] = state.Value
	return nil
}

// Returns a State's value from Database with the given key
// Or an error if a matching State does not exist
func (m *MapImpl) GetStateValue(key string) (string, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if val, ok := m.states[key]; ok {
		return val, nil
	} else {
		return "", errors.Errorf("Unable to locate state for key %s", key)
	}
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

// Update the address fields for the Node with the given id
func (m *MapImpl) UpdateNodeAddresses(id *id.ID, nodeAddr, gwAddr string) error {
	m.mut.Lock()
	defer m.mut.Unlock()

	for _, v := range m.nodes {
		if bytes.Compare(v.Id, id.Marshal()) == 0 {
			v.GatewayAddress = gwAddr
			v.ServerAddress = nodeAddr
			return nil
		}
	}

	return errors.Errorf("unable to update addresses for %s", id.String())
}
