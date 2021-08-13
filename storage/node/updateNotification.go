////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package node

// Contains the node's update notification object

import (
	"git.xx.network/elixxir/comms/mixmessages"
	"git.xx.network/elixxir/primitives/current"
	"git.xx.network/xx_network/primitives/id"
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
	ClientErrors []*mixmessages.ClientError
}
