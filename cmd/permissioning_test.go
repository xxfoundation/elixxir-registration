package cmd

import (
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/elixxir/registration/testkeys"
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

//Happy path: tests that the function loads active and banned nodes into the maps
func TestLoadAllRegisteredNodes(t *testing.T) {
	//region Database setup
	// Create a database to store some nodes into
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Error(err)
	}

	//Create reg codes and populate the database
	infos := make([]node.Info, 0)
	infos = append(infos, node.Info{RegCode: "AAAA", Order: "0"},
		node.Info{RegCode: "BBBB", Order: "1"},
		node.Info{RegCode: "CCCC", Order: "2"})
	storage.PopulateNodeRegistrationCodes(infos)
	//endregion

	//region Mock node setup
	// Get TLS cert
	crt, err := utils.ReadFile(testkeys.GetCACertPath())

	// Create a new ID and store a new active node into the database
	activeNodeId := id.NewIdFromUInt(0, id.Node, t)
	err = storage.PermissioningDb.RegisterNode(activeNodeId, []byte("test1"), "AAAA", "0.0.0.0", string(crt),
		"0.0.0.0", string(crt))
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1)

	// Create a new ID and store a new *banned* node into the database
	bannedNodeId := id.NewIdFromUInt(1, id.Node, t)
	err = storage.PermissioningDb.RegisterNode(bannedNodeId, []byte("test2"), "BBBB", "0.0.0.0", string(crt),
		"0.0.0.0", string(crt))
	if err != nil {
		t.Error(err)
	}
	time.Sleep(1)

	// Create a new ID and store a new *banned* node into the database
	altNodeID := id.NewIdFromString("alt", id.Node, t)
	err = storage.PermissioningDb.RegisterNode(altNodeID, []byte("test3"), "CCCC", "0.0.0.0", string(crt),
		"0.0.0.0", string(crt))
	if err != nil {
		t.Error(err)
	}

	permissioningMap := storage.PermissioningDb.Database.(*storage.MapImpl)
	err = permissioningMap.BannedNode(bannedNodeId, t)
	if err != nil {
		t.Error(err)
	}
	//endregion
	//region Test code
	// Create params for test registration server
	testParams := Params{
		CertPath:         testkeys.GetCACertPath(),
		KeyPath:          testkeys.GetCAKeyPath(),
		NdfOutputPath:    testkeys.GetNDFPath(),
		udbPubKeyPemPath: testkeys.GetUdbPubKeyPemPath(),
	}
	// Start registration server
	impl, err := StartRegistration(testParams)
	if err != nil {
		t.Error(err)
	}

	// Call to load all registered nodes from DB
	err = impl.LoadAllRegisteredNodes()
	if err != nil {
		t.Error("LoadAllRegisteredNodes returned an error: ", err)
	}
	//endregion

	//region Host map checking
	// TODO: there doesn't seem to be a way to get the number of nodes in the host map that's obvious to me
	// Check that the active node stuff is alright
	hmActiveNode, hmActiveNodeOk := impl.Comms.GetHost(activeNodeId)
	if !hmActiveNodeOk {
		t.Error("Getting active node from host map did not return okay.")
	}
	if !hmActiveNode.GetId().Cmp(activeNodeId) {
		t.Error("Unexpected node ID for node 0:\r\tGot: %i\r\tExpected: %i", hmActiveNode.GetId(), activeNodeId)
	}

	hmBannedNode, hmBannedNodeOk := impl.Comms.GetHost(bannedNodeId)
	if !hmBannedNodeOk {
		t.Error("Getting active node from host map did not return okay.")
	}
	if !hmBannedNode.GetId().Cmp(bannedNodeId) {
		t.Error("Unexpected node ID for node 0:\r\tGot: %i\r\tExpected: %i", hmBannedNode.GetId(), bannedNodeId)
	}

	//region Node map checking
	// Check that the nodes were added to the node map
	expected_nodes := 3
	nodeMapNodes := impl.State.GetNodeMap().GetNodeStates()
	if len(nodeMapNodes) != expected_nodes {
		t.Errorf("Unexpected number of nodes found in node map:\n\tGot: %d\n"+
			"\tExpected: %d", len(nodeMapNodes), expected_nodes)
	}
	def := impl.State.GetFullNdf().Get()
	id0, err := id.Unmarshal(def.Nodes[0].ID)
	if err != nil {
		t.Error("Failed to unmarshal ID")
	}
	if !id0.Cmp(activeNodeId) {
		t.Errorf("Unexpected node ID for node 0:\n\tGot: %d\n\tExpected: %d",
			nodeMapNodes[0].GetID(), activeNodeId)
	}

	id1, err := id.Unmarshal(def.Nodes[1].ID)
	if err != nil {
		t.Error("Failed to unmarshal ID")
	}
	if !id1.Cmp(bannedNodeId) {
		t.Errorf("Unexpected node ID for node 1:\n\tGot: %d\n\tExpected: %d",
			nodeMapNodes[1].GetID(), bannedNodeId)
	}

	id2, err := id.Unmarshal(def.Nodes[2].ID)
	if err != nil {
		t.Error("Failed to unmarshal ID")
	}
	if !id2.Cmp(altNodeID) {
		t.Errorf("Unexpected node ID for node 2:\n\tGot: %d\n\tExpected: %d",
			nodeMapNodes[2].GetID(), altNodeID)
	}

	banned := 0
	for _, n := range nodeMapNodes {
		if n.GetStatus() == node.Banned {
			banned++
		}
	}
	if banned != 1 {
		t.Error("Should only be one banned node")
	}
	//endregion

	// TODO: check servers get a valid NDF
	// Why? When I first made this code, it failed to add the nodes from the database into the NDF. Ideally this
	// would've been caught in testing, but I hadn't thought about that. It does seem like something pertinent to test
	// but at the time of me writing this code, we don't have the time to really do that.

	//region Cleanup
	// Shutdown registration
	//impl.Comms.Shutdown()
	//time.Sleep(10*time.Second)
	//endregion
}
