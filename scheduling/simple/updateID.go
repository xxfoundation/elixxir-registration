////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////
package simple

func NewUpdateID(start uint64) *UpdateID {
	updateID := UpdateID(start)
	return &updateID
}

type UpdateID uint64

func (i *UpdateID) Next() uint64 {
	old := *i
	*i += 1
	return uint64(old)
}

func (i *UpdateID) Get() uint64 {
	return uint64(*i)
}
