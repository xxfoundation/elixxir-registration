package simple

import "gitlab.com/elixxir/primitives/id"

func NewRoundID(start id.Round) *RoundID {
	roundID := RoundID(start)
	return &roundID
}

type RoundID id.Round

func (r *RoundID) Next() id.Round {
	old := *r
	*r += 1
	return id.Round(old)
}

func (r *RoundID) Get() id.Round {
	return id.Round(*r)
}
