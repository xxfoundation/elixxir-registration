package simple

import (
	"crypto/rand"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage"
	"strconv"
	"testing"
)

func TestCreateRound(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: false,
	}

	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey)
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	nodeList := make([]*id.Node, testParams.TeamSize)
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nodeList[i] = id.NewNodeFromUInt(i, t)
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i)))
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
	}

	testPool := &waitingPoll{
		pool:     nodeList,
		position: int(testParams.TeamSize),
	}

	roundID := NewRoundID(0)
	updateID := NewUpdateID(0)

	err = createRound(testParams, testPool, roundID, updateID, testState)
	if err != nil {
		t.Errorf("Happy path of createRound failed: %v", err)
	}
}
