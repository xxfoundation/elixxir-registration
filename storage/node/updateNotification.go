////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

// Contains the node's update notification object

import (
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/xx_network/primitives/id"
)

// UpdateNotification structure used to notify the control thread that the
// round state has updated.
type UpdateNotification struct {
	Node         *id.ID
	FromStatus   Status
	ToStatus     Status
	FromActivity current.Activity
	ToActivity   current.Activity
	Error        *mixmessages.RoundError
}
