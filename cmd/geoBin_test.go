////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/oschwald/geoip2-golang"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Tests that RegistrationImpl.setNodeSequence assigns a random bin to a node when
// randomGeoBinning is set.
func TestRegistrationImpl_setNodeBin_RandomBin(t *testing.T) {
	// Create a map database
	var err error
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Fatalf("Failed to create new database: %+v", err)
	}

	// Add an application to it
	testID := id.NewIdFromUInt(0, id.Node, t)
	err = storage.PermissioningDb.InsertApplication(
		&storage.Application{}, &storage.Node{Code: "AAAA"})
	if err != nil {
		t.Fatalf("Failed to insert application: %+v", err)
	}

	// Register a node
	err = storage.PermissioningDb.RegisterNode(testID, nil, "AAAA", "", "", "", "")
	if err != nil {
		t.Fatalf("Failed to register a node: %+v", err)
	}

	// Make a new state map and add the node to it
	stateMap := node.NewStateMap()
	err = stateMap.AddNode(testID, "", "", "", 0)
	if err != nil {
		t.Fatalf("Failed to add a node to the state map: %+v", err)
	}

	// Save node's bin before change to compare later
	nodeDB, err := storage.PermissioningDb.GetNodeById(testID)
	if err != nil {
		t.Fatalf("Failed to get node: %+v", err)
	}
	oldNodeBin := nodeDB.Sequence

	// Create a RegistrationImpl with randomGeoBinning enabled
	impl := &RegistrationImpl{params: &Params{randomGeoBinning: true}}

	// Expected to generate random bin
	err = impl.setNodeSequence(stateMap.GetNode(testID))
	if err != nil {
		t.Errorf("setNodeSequence returned an error: %+v", err)
	}

	// Get node's new bin
	nodeDB, err = storage.PermissioningDb.GetNodeById(testID)
	if err != nil {
		t.Fatalf("Failed to get node: %+v", err)
	}

	if nodeDB.Sequence == oldNodeBin {
		t.Errorf("setNodeSequence failed to modify the node's bin."+
			"\nold bin: %s\nnew bin: %s", oldNodeBin, nodeDB.Sequence)
	}

	ordering := stateMap.GetNode(testID).GetOrdering()
	if ordering == oldNodeBin {
		t.Errorf("setNodeSequence failed to modify the node's ordering."+
			"\nold bin: %s\nnew bin: %s", oldNodeBin, ordering)
	}
}

// Tests that RegistrationImpl.setNodeSequence assigns the correct geographic bin for
// the IP address.
func TestRegistrationImpl_setNodeBin_GeoIP2DB(t *testing.T) {
	var err error

	// Create registration impl
	impl := &RegistrationImpl{params: &Params{randomGeoBinning: false}}

	// Setup a reader with the testing database
	impl.geoIPDB, err = geoip2.Open("../testkeys/GeoIP2-City-Test.mmdb")
	if err != nil {
		t.Fatalf("Failed to open GeoIP2 database file: %+v", err)
	}
	impl.geoIPDBStatus.ToRunning()

	// Create a map database
	storage.PermissioningDb, _, err = storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Fatalf("Failed to create new database: %+v", err)
	}

	// Add an application to it
	testID := id.NewIdFromUInt(0, id.Node, t)
	err = storage.PermissioningDb.InsertApplication(
		&storage.Application{}, &storage.Node{Code: "AAAA"})
	if err != nil {
		t.Fatalf("Failed to insert application: %+v", err)
	}

	// Register a node
	err = storage.PermissioningDb.RegisterNode(testID, nil, "AAAA", "", "", "", "")
	if err != nil {
		t.Fatalf("Failed to register a node: %+v", err)
	}

	// Make a new state map and add the node to it
	stateMap := node.NewStateMap()
	err = stateMap.AddNode(testID, "", "202.196.224.6:2400", "", 0)
	if err != nil {
		t.Fatalf("Failed to add a node to the state map: %+v", err)
	}

	// Call setNodeSequence
	err = impl.setNodeSequence(stateMap.GetNode(testID))
	if err != nil {
		t.Errorf("setNodeSequence returned an error: %+v", err)
	}

	// Get node's new bin
	nodeDB, err := storage.PermissioningDb.GetNodeById(testID)
	if err != nil {
		t.Fatalf("Failed to get node: %+v", err)
	}

	if nodeDB.Sequence != "Asia" {
		t.Errorf("setNodeSequence failed to set the expected bin."+
			"\nexpected: %s\nreceived: %s", "Asia", nodeDB.Sequence)
	}

	ordering := stateMap.GetNode(testID).GetOrdering()
	if ordering != "Asia" {
		t.Errorf("setNodeSequence failed to set the state ordering to the expected bin."+
			"\nexpected: %s\nreceived: %s", "Asia", ordering)
	}
}

// Panic path: test that RegistrationImpl.setNodeSequence panics when neither a
// GeoIP2 reader is supplied nor is randomGeoBinning set.
func TestRegistrationImpl_setNodeBin_NoFlags(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("setNodeSequence failed to panic when neither flag was set.")
		}
	}()

	impl := &RegistrationImpl{}
	_ = impl.setNodeSequence(&node.State{})
}
