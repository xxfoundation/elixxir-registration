package scheduling

import (
	"crypto/rand"
	"github.com/katzenpost/core/crypto/eddsa"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/crypto/signature/rsa"
	"gitlab.com/xx_network/primitives/id"
	mathRand "math/rand"

	"strconv"
	"testing"
	"time"
)

// Happy path
func TestCreateRound(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            9,
		BatchSize:           32,
		RandomOrdering:      true,
		Threshold:           1,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ecPrivKey, err := eddsa.NewKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate elliptic private key:\n%v", err)
	}

	testState, err := storage.NewState(privKey, ecPrivKey, 8, "")
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
		err := testState.GetNodeMap().AddNode(nodeList[i], "Americas", "", "", 0)
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

	_, err = createSecureRound(testParams, testpool, roundID, testState)
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
		RandomOrdering:      true,
		Threshold:           5,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ecPrivKey, err := eddsa.NewKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate elliptic private key:\n%v", err)
	}

	testState, err := storage.NewState(privKey, ecPrivKey, 8, "")
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

	_, err = createSecureRound(testParams, testpool, roundID, testState)
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
		RandomOrdering:      true,
		Threshold:           25,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ecPrivKey, err := eddsa.NewKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate elliptic private key:\n%v", err)
	}

	testState, err := storage.NewState(privKey, ecPrivKey, 8, "")
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

	_, err = createSecureRound(testParams, testpool, roundID, testState)
	if err != nil {
		return
	}

	t.Errorf("Expected error path: Number of nodes in pool" +
		" shouldn't be enough for threshold")

}

// Test that a team of 8 nodes, each in a different region
// is assembled into a round with an efficient order
func TestCreateRound_EfficientTeam_AllRegions(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            8,
		BatchSize:           32,
		RandomOrdering:      true,
		Threshold:           2,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ecPrivKey, err := eddsa.NewKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate elliptic private key:\n%v", err)
	}

	testState, err := storage.NewState(privKey, ecPrivKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.ID, testParams.TeamSize)
	nodeStateList := make([]*node.State, testParams.TeamSize)

	// Craft regions for nodes
	regions := []string{"Americas", "WesternEurope", "CentralEurope",
		"EasternEurope", "MiddleEast", "Africa", "Russia", "Asia"}

	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid
		err := testState.GetNodeMap().AddNode(nodeList[i], regions[i], "", "", 0)
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

	start := time.Now()
	testProtoRound, err := createSecureRound(testParams, testpool, roundID, testState)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	duration := time.Now().Sub(start)
	t.Logf("CreateRound took: %v\n", duration)

	expectedDuration := int64(35)

	if duration.Milliseconds() > expectedDuration {
		t.Errorf("Warning, creating round for a team of 8 took longer than expected."+
			"\n\tExpected: ~%v ms"+
			"\n\tReceived: %v ms", expectedDuration, duration)
	}

	var regionOrder []int
	var regionOrderStr []string
	for _, n := range testProtoRound.NodeStateList {
		order, _ := getRegion(n.GetOrdering())
		region := n.GetOrdering()
		regionOrder = append(regionOrder, order)
		regionOrderStr = append(regionOrderStr, region)
	}

	t.Log("Team order outputted by CreateRound: ", regionOrderStr)

	// Go though the regions, checking for any long jumps
	validRegionTransitions := newTransitions()
	longTransitions := uint32(0)
	for i, thisRegion := range regionOrder {
		// Get the next region to  see if it's a long distant jump
		nextRegion := regionOrder[(i+1)%len(regionOrder)]
		if !validRegionTransitions.isValidTransition(thisRegion, nextRegion) {
			longTransitions++
		}

	}

	t.Logf("Amount of long distant jumps: %v", longTransitions)

	// Check that the long jumps does not exceed over half the jumps
	if longTransitions > testParams.TeamSize/2+1 {
		t.Errorf("Number of long distant transitions beyond acceptable amount!"+
			"\n\tAcceptable long distance transitions: %v"+
			"\n\tReceived long distance transitions: %v", testParams.TeamSize/2+1, longTransitions)
	}

}

// Test that a team of 8 nodes from random regions,
// is assembled into a round with an efficient order
func TestCreateRound_EfficientTeam_RandomRegions(t *testing.T) {
	testpool := NewWaitingPool()

	// Build scheduling params
	testParams := Params{
		TeamSize:            8,
		BatchSize:           32,
		RandomOrdering:      true,
		Threshold:           2,
		NodeCleanUpInterval: 3,
	}

	// Build network state
	privKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	ecPrivKey, err := eddsa.NewKeypair(rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate elliptic private key:\n%v", err)
	}

	testState, err := storage.NewState(privKey, ecPrivKey, 8, "")
	if err != nil {
		t.Errorf("Failed to create test state: %v", err)
		t.FailNow()
	}

	// Build the nodes
	nodeList := make([]*id.ID, testParams.TeamSize*2)
	nodeStateList := make([]*node.State, testParams.TeamSize*2)

	// Craft regions for nodes
	regions := []string{"Americas", "WesternEurope", "CentralEurope",
		"EasternEurope", "MiddleEast", "Africa", "Russia", "Asia"}

	// Populate the pool with 2x the team size
	for i := uint64(0); i < uint64(len(nodeList)); i++ {
		// Randomize the regions of the nodes
		index := mathRand.Intn(8)

		// Generate a test id
		nid := id.NewIdFromUInt(i, id.Node, t)
		nodeList[i] = nid

		// Add the node to that node map
		// Place the node in a random region
		err := testState.GetNodeMap().AddNode(nodeList[i], regions[index], "", "", 0)
		if err != nil {
			t.Errorf("Couldn't add node: %v", err)
			t.FailNow()
		}

		// Add the node to the pool
		nodeState := testState.GetNodeMap().GetNode(nid)
		nodeStateList[i] = nodeState
		testpool.Add(nodeState)
	}

	roundID, err := testState.IncrementRoundID()
	if err != nil {
		t.Errorf("IncrementRoundID() failed: %+v", err)
	}

	//  Create the protoround
	start := time.Now()
	testProtoRound, err := createSecureRound(testParams, testpool, roundID, testState)
	if err != nil {
		t.Errorf("Error in happy path: %v", err)
	}

	duration := time.Now().Sub(start)
	expectedDuration := int64(40)

	// Check that it did not take an excessive amount of time
	// to create the round
	if duration.Milliseconds() > expectedDuration {
		t.Errorf("Warning, creating round for a team of 8 took longer than expected."+
			"\n\tExpected: ~%v ms"+
			"\n\tReceived: %v ms", expectedDuration, duration)
	}

	// Parse the order of the regions
	// one for testing and one for logging
	var regionOrder []int
	var regionOrderStr []string
	for _, n := range testProtoRound.NodeStateList {
		order, _ := getRegion(n.GetOrdering())
		region := n.GetOrdering()
		regionOrder = append(regionOrder, order)
		regionOrderStr = append(regionOrderStr, region)
	}

	// Output the teaming order to the log in human readable format
	t.Log("Team order outputted by CreateRound: ", regionOrderStr)

	// Measure the amount of longer than necessary jumps
	validRegionTransitions := newTransitions()
	longTransitions := uint32(0)
	for i, thisRegion := range regionOrder {
		// Get the next region to  see if it's a long distant jump
		nextRegion := regionOrder[(i+1)%len(regionOrder)]
		if !validRegionTransitions.isValidTransition(thisRegion, nextRegion) {
			longTransitions++
		}

	}

	t.Logf("Amount of long distant jumps: %v", longTransitions)

	// Check that the long distant jumps do not exceed half the jumps
	if longTransitions > testParams.TeamSize/2+1 {
		t.Errorf("Number of long distant transitions beyond acceptable amount!"+
			"\n\tAcceptable long distance transitions: %v"+
			"\n\tReceived long distance transitions: %v", testParams.TeamSize/2+1, longTransitions)
	}

}

// Based on the control state logic used for rounds. Based on the map
// discerned from internet cable maps
type regionTransition [8]regionTransitionValidation

// Transitional information used for each region
type regionTransitionValidation struct {
	from [8]bool
}

// Create the valid jumps for each region
func newRegionTransitionValidation(from ...int) regionTransitionValidation {
	tv := regionTransitionValidation{}

	for _, f := range from {
		tv.from[f] = true
	}

	return tv
}

// Valid transitions are defined as region jumps that are not long distant
// long distant is defined by internet cable maps. It was defined
// in a undirected graph of what are good internet connections
func newTransitions() regionTransition {
	t := regionTransition{}
	t[Americas] = newRegionTransitionValidation(Americas, Asia, WesternEurope)
	t[WesternEurope] = newRegionTransitionValidation(WesternEurope, Americas, Africa, CentralEurope)
	t[CentralEurope] = newRegionTransitionValidation(CentralEurope, Africa, MiddleEast, EasternEurope, WesternEurope)
	t[EasternEurope] = newRegionTransitionValidation(EasternEurope, MiddleEast, Russia, CentralEurope)
	t[MiddleEast] = newRegionTransitionValidation(MiddleEast, EasternEurope, Asia, CentralEurope)
	t[Africa] = newRegionTransitionValidation(Africa, WesternEurope, CentralEurope)
	t[Russia] = newRegionTransitionValidation(Russia, Asia, EasternEurope)
	t[Asia] = newRegionTransitionValidation(Asia, Americas, MiddleEast, Russia)

	return t
}

// IsValidTransition checks the transitionValidation to see if
//  the attempted transition is valid
func (r regionTransition) isValidTransition(from, to int) bool {
	return r[to].from[from]
}
