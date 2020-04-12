package simple

import (
	"crypto/rand"
	"gitlab.com/elixxir/comms/connect"
	"gitlab.com/elixxir/crypto/signature/rsa"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/states"
	"gitlab.com/elixxir/registration/storage"
	"reflect"
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
	updateID := NewUpdateID(1)

	err = createRound(testParams, testPool, roundID, updateID, testState)
	if err != nil {
		t.Errorf("Happy path of createRound failed: %v", err)
	}

	if testState.GetRoundMap().GetRound(0).GetRoundState() != states.PRECOMPUTING {
		t.Errorf("In unexpected state after round creation: %v",
			testState.GetRoundMap().GetRound(0).GetRoundState())
	}

}

func TestCreateRound_BadOrdering(t *testing.T) {
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
		// Input an invalid ordering to node
		err := testState.GetNodeMap().AddNode(nodeList[i], "BadNumber")
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
	updateID := NewUpdateID(1)

	// Invalid ordering will cause this to fail
	err = createRound(testParams, testPool, roundID, updateID, testState)
	if err != nil {
		return
	}

	t.Errorf("Expected error case: passed in an ordering to nodes which were not numbers should result " +
		"in an error")

}

func TestCreateRound_RandomOrdering(t *testing.T) {
	testParams := Params{
		TeamSize:       5,
		BatchSize:      32,
		RandomOrdering: true,
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

	initialTopology := connect.NewCircuit(nodeList)

	testPool := &waitingPoll{
		pool:     nodeList,
		position: int(testParams.TeamSize),
	}

	roundID := NewRoundID(0)
	updateID := NewUpdateID(1)

	err = createRound(testParams, testPool, roundID, updateID, testState)
	if err != nil {
		t.Errorf("Happy path of createRound failed: %v", err)
	}

	resultingTopology := testState.GetRoundMap().GetRound(0).GetTopology()

	if reflect.DeepEqual(initialTopology, resultingTopology) {
		t.Errorf("Highly unlikely initial topology identical to resulting after shuffling. " +
			"Possile shuffling is broken")
	}
}
