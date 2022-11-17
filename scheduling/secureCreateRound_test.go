////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package scheduling

import (
	"crypto/rand"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	mathRand "math/rand"

	"strconv"
	"testing"
)

// Happy path
func TestCreateRound(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            9,
		BatchSize:           32,
		Threshold:           0.3,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], "US", "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testpool.Add(nodeState)
	}

	roundID, err := testState.GetRoundID()
	if err != nil {
		t.Errorf(err.Error())
	}

	prng := mathRand.New(mathRand.NewSource(42))

	_, err = createSecureRound(testParams, testpool, int(testParams.Threshold*float64(testParams.TeamSize)), roundID, testState, prng)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}
}

func TestCreateRound_Error_NotEnoughForTeam(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            9,
		BatchSize:           32,
		Threshold:           5,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)

	// Do not make a teamsize amount of nodes
	for i := uint64(0); i < 5; i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i%5)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testpool.Add(nodeState)
	}

	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}
	prng := mathRand.New(mathRand.NewSource(42))

	_, err = createSecureRound(testParams, testpool, int(testParams.Threshold*float64(testParams.TeamSize)), roundID, testState, prng)
	if err != nil {
		return
	}

	t.Errorf("Expected error path: Number of nodes in pool" +
		" shouldn't be enough for a team size")

}

func TestCreateRound_Error_NotEnoughForThreshold(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            9,
		BatchSize:           32,
		Threshold:           25,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)

	testState, err := storage.NewState(privKey, 8, "", "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)

	// Do not make a teamsize amount of nodes
	for i := uint64(0); i < 5; i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], strconv.Itoa(int(i%5)), "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testpool.Add(nodeState)
	}

	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}
	prng := mathRand.New(mathRand.NewSource(42))

	_, err = createSecureRound(testParams, testpool, int(testParams.Threshold*float64(testParams.TeamSize)), roundID, testState, prng)
	if err != nil {
		return
	}

	t.Errorf("Expected error path: Number of nodes in pool" +
		" shouldn't be enough for threshold")

}
