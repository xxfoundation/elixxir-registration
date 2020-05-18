package node

import (
	"gitlab.com/elixxir/primitives/current"
	"gitlab.com/elixxir/primitives/id"
)

// UpdateNotification structure used to notify the control thread that the
// round state has updated.
type UpdateNotification struct {
	Node         *id.ID
	FromStatus   Status
	ToStatus     Status
	FromActivity current.Activity
	ToActivity   current.Activity
}


