////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

//this file defines the structure which creates and tracks the RoundID

package scheduling

//tracks the round ID. Only allows itself to incremented forward by 1
import "gitlab.com/elixxir/primitives/id"

//creates teh RoundID structure
func NewRoundID(start id.Round) *RoundID {
	roundID := RoundID(start)
	return &roundID
}

//defines the RoundID type
type RoundID id.Round

//Returns the current count and increments to the next one
func (r *RoundID) Next() id.Round {
	old := *r
	*r += 1
	return id.Round(old)
}

//returns the current count
func (r *RoundID) Get() id.Round {
	return id.Round(*r)
}
