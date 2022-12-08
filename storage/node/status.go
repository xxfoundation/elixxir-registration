////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package node

// Contains the enumeration of a node.State's status field.
//This differs from state as it is not driven by the node's
//internal activities, but rather the status as relates to
// the network.

type Status uint8

const (
	Unregistered = Status(iota) // Default state, equivalent to NULL
	Active                      // Operational, active Node which will be considered for team
	Inactive                    // Inactive for a certain amount of time, not considered for teams
	Banned                      // Stop any teams and ban from teams until manually overridden
)

// Stringer for the status type
func (s Status) String() string {
	switch s {
	case Unregistered:
		return "Unregistered"
	case Active:
		return "Active"
	case Inactive:
		return "Inactive"
	case Banned:
		return "Banned"
	default:
		return "Unknown"
	}
}
