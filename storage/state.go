////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles network state tracking and control

package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	pb "gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/ndf"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/storage/round"
)

const updateBufferLength = 1000

// NetworkState structure used for keeping track of NDF and Round state.
type NetworkState struct {
	// NetworkState parameters
	privateKey *rsa.PrivateKey

	// Round state
	rounds       *round.StateMap
	roundUpdates *dataStructures.Updates
	update       chan *NodeUpdateNotification // For triggering updates to top level

	// Node NetworkState
	nodes *node.StateMap

	// NDF state
	partialNdf *dataStructures.Ndf
	fullNdf    *dataStructures.Ndf
}

// NodeUpdateNotification structure used to notify the control thread that the
// round state has updated.
type NodeUpdateNotification struct {
	Node *id.Node
	From current.Activity
	To   current.Activity
}

// NewState returns a new NetworkState object.
func NewState(pk *rsa.PrivateKey) (*NetworkState, error) {
	fullNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		return nil, err
	}
	partialNdf, err := dataStructures.NewNdf(&ndf.NetworkDefinition{})
	if err != nil {
		return nil, err
	}

	state := &NetworkState{
		rounds:       round.NewStateMap(),
		roundUpdates: dataStructures.NewUpdates(),
		update:       make(chan *NodeUpdateNotification, updateBufferLength),
		nodes:        node.NewStateMap(),
		fullNdf:      fullNdf,
		partialNdf:   partialNdf,
		privateKey:   pk,
	}

	// Insert dummy update
	err = state.AddRoundUpdate(0, &pb.RoundInfo{})
	if err != nil {
		return nil, err
	}
	return state, nil
}

// GetFullNdf returns the full NDF.
func (s *NetworkState) GetFullNdf() *dataStructures.Ndf {
	return s.fullNdf
}

// GetPartialNdf returns the partial NDF.
func (s *NetworkState) GetPartialNdf() *dataStructures.Ndf {
	return s.partialNdf
}

// GetUpdates returns all of the updates after the given ID.
func (s *NetworkState) GetUpdates(id int) ([]*pb.RoundInfo, error) {
	return s.roundUpdates.GetUpdates(id)
}

// AddRoundUpdate creates a copy of the round before inserting it into
// roundUpdates.
func (s *NetworkState) AddRoundUpdate(updateID uint64, round *pb.RoundInfo) error {
	roundCopy := &pb.RoundInfo{
		ID:         round.GetID(),
		UpdateID:   updateID,
		State:      round.GetState(),
		BatchSize:  round.GetBatchSize(),
		Topology:   round.GetTopology(),
		Timestamps: round.GetTimestamps(),
	}

	err := signature.Sign(roundCopy, s.privateKey)
	if err != nil {
		return errors.WithMessagef(err, "Could not add round update %v "+
			"due to failed signature", roundCopy.UpdateID)
	}

	jww.DEBUG.Printf("Round state updated to %s",
		states.Round(roundCopy.State))

	return s.roundUpdates.AddRound(roundCopy)
}

// UpdateNdf updates internal NDF structures with the specified new NDF.
func (s *NetworkState) UpdateNdf(newNdf *ndf.NetworkDefinition) (err error) {
	// Build NDF comms messages
	fullNdfMsg := &pb.NDF{}
	fullNdfMsg.Ndf, err = newNdf.Marshal()
	if err != nil {
		return
	}
	partialNdfMsg := &pb.NDF{}
	partialNdfMsg.Ndf, err = newNdf.StripNdf().Marshal()
	if err != nil {
		return
	}

	// Sign NDF comms messages
	err = signature.Sign(fullNdfMsg, s.privateKey)
	if err != nil {
		return
	}
	err = signature.Sign(partialNdfMsg, s.privateKey)
	if err != nil {
		return
	}

	// Assign NDF comms messages
	err = s.fullNdf.Update(fullNdfMsg)
	if err != nil {
		return err
	}
	return s.partialNdf.Update(partialNdfMsg)
}

// GetPrivateKey returns the server's private key.
func (s *NetworkState) GetPrivateKey() *rsa.PrivateKey {
	return s.privateKey
}

// GetRoundMap returns the map of rounds.
func (s *NetworkState) GetRoundMap() *round.StateMap {
	return s.rounds
}

// GetNodeMap returns the map of nodes.
func (s *NetworkState) GetNodeMap() *node.StateMap {
	return s.nodes
}

// NodeUpdateNotification sends a notification to the control thread of an
// update to a nodes state.
func (s *NetworkState) NodeUpdateNotification(node *id.Node, from, to current.Activity) error {
	nun := NodeUpdateNotification{
		Node: node,
		From: from,
		To:   to,
	}

	select {
	case s.update <- &nun:
		return nil
	default:
		return errors.New("Could not send update notification")
	}
}

// GetNodeUpdateChannel returns a channel to receive node updates on.
func (s *NetworkState) GetNodeUpdateChannel() <-chan *NodeUpdateNotification {
	return s.update
}
