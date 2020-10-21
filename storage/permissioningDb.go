////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for nodes

package storage

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
)

// Inserts the given State into database if it does not exist
// Or updates the database State if its value does not match the given State
func (m *DatabaseImpl) UpsertState(state *State) error {
	// Build a transaction to prevent race conditions
	return m.db.Transaction(func(tx *gorm.DB) error {
		// Initialize variable for returning existing value from the database
		oldState := &State{}

		// Attempt to insert state into the database,
		// or if it already exists, replace oldState with the database value
		err := tx.FirstOrCreate(oldState, state).Error
		if err != nil {
			return err
		}

		// If oldState is already present in the database, overwrite it with state
		if oldState.Value != state.Value {
			return tx.Save(state).Error
		}

		// Commit
		return nil
	})
}

// Returns a State's value from database with the given key
// Or an error if a matching State does not exist
func (m *DatabaseImpl) GetStateValue(key string) (string, error) {
	result := &State{Key: key}
	err := m.db.Take(result).Error
	return result.Value, err
}

// Insert NodeMetric object
func (m *DatabaseImpl) InsertNodeMetric(metric *NodeMetric) error {
	jww.TRACE.Printf("Attempting to insert node metric: %+v", metric)
	return m.db.Create(metric).Error
}

// Insert RoundError object
func (m *DatabaseImpl) InsertRoundError(roundId id.Round, errStr string) error {
	roundErr := &RoundError{
		RoundMetricId: uint64(roundId),
		Error:         errStr,
	}
	jww.DEBUG.Printf("Attempting to insert round error: %+v", roundErr)
	return m.db.Create(roundErr).Error
}

// Insert RoundMetric object with associated topology
func (m *DatabaseImpl) InsertRoundMetric(metric *RoundMetric, topology [][]byte) error {

	// Build the Topology
	metric.Topologies = make([]Topology, len(topology))
	for i, nodeIdBytes := range topology {
		nodeId, err := id.Unmarshal(nodeIdBytes)
		if err != nil {
			return errors.New(err.Error())
		}
		topologyObj := Topology{
			NodeId: nodeId.Bytes(),
			Order:  uint8(i),
		}
		metric.Topologies[i] = topologyObj
	}

	// Save the RoundMetric
	jww.DEBUG.Printf("Attempting to insert round metric: %+v", metric)
	return m.db.Create(metric).Error
}
