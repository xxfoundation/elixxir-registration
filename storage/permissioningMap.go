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
	"time"
)

// Inserts the given State into Storage if it does not exist
// Or updates the Database State if its value does not match the given State
func (m *MapImpl) UpsertState(state *State) error {
	jww.TRACE.Printf("Attempting to insert State into Map: %+v", state)

	m.mut.Lock()
	defer m.mut.Unlock()

	m.states[state.Key] = state.Value
	return nil
}

// Returns a State's value from Storage with the given key
// Or an error if a matching State does not exist
func (m *MapImpl) GetStateValue(key string) (string, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if val, ok := m.states[key]; ok {
		jww.TRACE.Printf("Obtained State from Map: %+v", val)
		return val, nil
	}

	// NOTE: Other code depends on this error string
	return "", errors.Errorf("Unable to locate state for key %s", key)
}

// Insert new NodeMetric object into Storage
func (m *MapImpl) InsertNodeMetric(metric *NodeMetric) error {
	jww.TRACE.Printf("Attempting to insert NodeMetric into Map: %+v", metric)

	m.mut.Lock()
	defer m.mut.Unlock()

	// Auto-increment key
	m.nodeMetricCounter += 1

	// Add to map
	metric.Id = m.nodeMetricCounter
	m.nodeMetrics[m.nodeMetricCounter] = metric
	return nil
}

// Insert new RoundError object into Storage
func (m *MapImpl) InsertRoundError(roundId id.Round, errStr string) error {
	rid := uint64(roundId)
	roundErr := RoundError{
		Id:            0, // Currently useless in MapImpl
		RoundMetricId: rid,
		Error:         errStr,
	}

	jww.TRACE.Printf("Attempting to insert RoundError into Map: %+v", roundErr)
	m.mut.Lock()
	m.roundMetrics[rid].RoundErrors = append(m.roundMetrics[rid].RoundErrors, roundErr)
	m.mut.Unlock()
	return nil
}

// Insert new RoundMetric object with associated topology into Storage
func (m *MapImpl) InsertRoundMetric(metric *RoundMetric, topology [][]byte) error {

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
	jww.TRACE.Printf("Attempting to insert RoundMetric into Map: %+v", metric)
	m.mut.Lock()
	m.roundMetrics[metric.Id] = metric
	m.mut.Unlock()
	return nil
}

// Returns newest (and largest, by implication) EphemeralLength from Storage
func (m *MapImpl) GetLatestEphemeralLength() (*EphemeralLength, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if len(m.ephemeralLengths) == 0 {
		return nil, errors.Errorf("Unable to locate any EphemeralLengths")
	}

	largest := uint8(0)
	for k := range m.ephemeralLengths {
		if k > largest {
			largest = k
		}
	}
	result := m.ephemeralLengths[largest]
	jww.TRACE.Printf("Obtained latest EphemeralLength from Map: %+v", result)
	return result, nil
}

// Returns all EphemeralLength from Storage
func (m *MapImpl) GetEphemeralLengths() ([]*EphemeralLength, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if len(m.ephemeralLengths) == 0 {
		return nil, errors.Errorf("Unable to locate any EphemeralLengths")
	}

	result := make([]*EphemeralLength, len(m.ephemeralLengths))
	i := 0
	for _, v := range m.ephemeralLengths {
		result[i] = v
		i++
	}
	jww.TRACE.Printf("Obtained EphemeralLengths from Map: %+v", result)
	return result, nil
}

// Insert new EphemeralLength into Storage
func (m *MapImpl) InsertEphemeralLength(length *EphemeralLength) error {
	jww.TRACE.Printf("Attempting to insert EphemeralLength into Map: %+v", length)

	m.mut.Lock()
	defer m.mut.Unlock()

	if m.ephemeralLengths[length.Length] != nil {
		return errors.Errorf("ephemeral length %d already exists", length.Length)
	}

	m.ephemeralLengths[length.Length] = length
	return nil
}

// Get the first round that is timestamped after the given cutoff
func (m *MapImpl) GetEarliestRound(cutoff time.Duration) (id.Round, error) {
	cutoffTs := time.Now().Add(-cutoff)
	minRound := &RoundMetric{}
	for _, v := range m.roundMetrics {
		if v.RealtimeEnd.After(cutoffTs) && (v.Id < minRound.Id || minRound.Id == 0) {
			minRound = v
		}
	}
	return id.Round(minRound.Id), nil
}

// Returns all GeoBin from Storage
func (m *MapImpl) getBins() ([]*GeoBin, error) {
	m.mut.Lock()
	defer m.mut.Unlock()

	if m.geographicBin == nil || len(m.geographicBin) == 0 {
		return nil, errors.Errorf("No geographic bins present in storage")
	}

	geoBins := make([]*GeoBin, 0, len(m.geographicBin))
	for countryCode, bin := range m.geographicBin {
		geoBin := &GeoBin{Bin: bin, Country: countryCode}
		geoBins = append(geoBins, geoBin)
	}

	return geoBins, nil
}
