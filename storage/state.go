////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles network state tracking and control

package storage

import (
	"gitlab.com/elixxir/comms/network/dataStructures"
	"gitlab.com/elixxir/primitives/ndf"
)

// Used for keeping track of NDF and Round state
type State struct {
	partialNdf   *dataStructures.Ndf
	fullNdf      *dataStructures.Ndf
	RoundUpdates *dataStructures.Updates
	RoundData    *dataStructures.Data
}

// Given a full NDF, updates both of the internal NDF structures
func (s *State) UpdateNdf(newNdf *ndf.NetworkDefinition) (err error) {
	s.fullNdf, err = dataStructures.NewNdf(newNdf)
	if err != nil {
		return
	}

	s.partialNdf, err = dataStructures.NewNdf(newNdf.StripNdf())
	return
}

// Returns the full NDF
func (s *State) GetFullNdf() *dataStructures.Ndf {
	return s.fullNdf
}

// Returns the partial NDF
func (s *State) GetPartiallNdf() *dataStructures.Ndf {
	return s.partialNdf
}
