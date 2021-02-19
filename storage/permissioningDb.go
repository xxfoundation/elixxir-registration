////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the DatabaseImpl for permissioning-based functionality
//+build !stateless

package storage

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
)

// Inserts the given State into Storage if it does not exist
// Or updates the Database State if its value does not match the given State
func (d *DatabaseImpl) UpsertState(state *State) error {
	jww.TRACE.Printf("Attempting to insert State into DB: %+v", state)

	// Build a transaction to prevent race conditions
	return d.db.Transaction(func(tx *gorm.DB) error {
		// Make a copy of the provided state
		newState := *state

		// Attempt to insert state into the Database,
		// or if it already exists, replace state with the Database value
		err := tx.FirstOrCreate(state, &State{Key: state.Key}).Error
		if err != nil {
			return err
		}

		// If state is already present in the Database, overwrite it with newState
		if newState.Value != state.Value {
			return tx.Save(newState).Error
		}

		// Commit
		return nil
	})
}

// Returns a State's value from Storage with the given key
// Or an error if a matching State does not exist
func (d *DatabaseImpl) GetStateValue(key string) (string, error) {
	result := &State{Key: key}
	err := d.db.Take(result).Error
	jww.TRACE.Printf("Obtained State from DB: %+v", result)
	return result.Value, err
}

// Insert new NodeMetric object into Storage
func (d *DatabaseImpl) InsertNodeMetric(metric *NodeMetric) error {
	jww.TRACE.Printf("Attempting to insert NodeMetric into DB: %+v", metric)
	return d.db.Create(metric).Error
}

// Insert new RoundError object into Storage
func (d *DatabaseImpl) InsertRoundError(roundId id.Round, errStr string) error {
	roundErr := &RoundError{
		RoundMetricId: uint64(roundId),
		Error:         errStr,
	}
	jww.TRACE.Printf("Attempting to insert RoundError into DB: %+v", roundErr)
	return d.db.Create(roundErr).Error
}

// Insert new RoundMetric object with associated topology into Storage
func (d *DatabaseImpl) InsertRoundMetric(metric *RoundMetric, topology [][]byte) error {

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
	jww.TRACE.Printf("Attempting to insert RoundMetric into DB: %+v", metric)
	return d.db.Create(metric).Error
}

// Returns newest (and largest, by implication) EphemeralLength from Storage
func (d *DatabaseImpl) GetLatestEphemeralLength() (*EphemeralLength, error) {
	result := &EphemeralLength{}
	err := d.db.Last(result).Error
	jww.TRACE.Printf("Obtained latest EphemeralLength from DB: %+v", result)
	return result, err
}

// Returns all EphemeralLength from Storage
func (d *DatabaseImpl) GetEphemeralLengths() ([]*EphemeralLength, error) {
	var result []*EphemeralLength
	err := d.db.Find(&result).Error
	jww.TRACE.Printf("Obtained EphemeralLengths from DB: %+v", result)
	return result, err
}

// Insert new EphemeralLength into Storage
func (d *DatabaseImpl) InsertEphemeralLength(length *EphemeralLength) error {
	jww.TRACE.Printf("Attempting to insert EphemeralLength into DB: %+v", length)
	return d.db.Create(length).Error
}
