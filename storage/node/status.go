////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

type Status uint8

const (
	Unregistered = Status(iota) // Default state, equivalent to NULL
	Active                      // Operational, active Node which will be considered for team
	Inactive                    // Inactive for a certain amount of time, not considered for teams
	Banned                      // Stop any teams and ban from teams until manually overridden
)

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
