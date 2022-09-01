////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
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
	ClientErrors []*mixmessages.ClientError
}
