////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the DatabaseImpl for permissioning-based functionality

package storage

import (
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/xx_network/primitives/id"
)

// Inserts the given State into Database if it does not exist
// Or updates the Database State if its value does not match the given State
func (d *DatabaseImpl) UpsertState(state *State) error {
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

// Returns a State's value from Database with the given key
// Or an error if a matching State does not exist
func (d *DatabaseImpl) GetStateValue(key string) (string, error) {
	result := &State{Key: key}
	err := d.db.Take(result).Error
	return result.Value, err
}

// Insert NodeMetric object
func (d *DatabaseImpl) InsertNodeMetric(metric *NodeMetric) error {
	jww.TRACE.Printf("Attempting to insert node metric: %+v", metric)
	return d.db.Create(metric).Error
}

// Insert RoundError object
func (d *DatabaseImpl) InsertRoundError(roundId id.Round, errStr string) error {
	roundErr := &RoundError{
		RoundMetricId: uint64(roundId),
		Error:         errStr,
	}
	jww.DEBUG.Printf("Attempting to insert round error: %+v", roundErr)
	return d.db.Create(roundErr).Error
}

// Insert RoundMetric object with associated topology
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
	jww.DEBUG.Printf("Attempting to insert round metric: %+v", metric)
	return d.db.Create(metric).Error
}

// Update the address fields for the Node with the given id
func (d *DatabaseImpl) UpdateNodeAddresses(id *id.ID, nodeAddr, gwAddr string) error {
	newNode := &Node{
		Id:             id.Marshal(),
		ServerAddress:  nodeAddr,
		GatewayAddress: gwAddr,
	}
	return d.db.Model(newNode).Where("id = ?", newNode.Id).Updates(map[string]interface{}{
		"server_address":  nodeAddr,
		"gateway_address": gwAddr,
	}).Error
}
