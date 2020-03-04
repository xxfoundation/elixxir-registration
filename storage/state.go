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

//
type State struct {
	partialNdf   *dataStructures.Ndf
	fullNdf      *dataStructures.Ndf
	roundUpdates *dataStructures.Updates
	roundData    *dataStructures.Data
}

//
func (s *State) UpdateNdf(newNdf *ndf.NetworkDefinition) (err error) {
	s.fullNdf, err = dataStructures.NewNdf(newNdf)
	if err != nil {
		return
	}

	s.partialNdf, err = dataStructures.NewNdf(newNdf.StripNdf())
	return
}

//
func (s *State) GetFullNdf() *dataStructures.Ndf {
	return s.fullNdf
}

//
func (s *State) GetPartiallNdf() *dataStructures.Ndf {
	return s.partialNdf
}
