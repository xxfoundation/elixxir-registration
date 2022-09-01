////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"testing"
)

// Tests that RegistrationImpl.setNodeSequence assigns the correct geographic bin for
// the IP address.
func TestRegistrationImpl_setNodeBin_GeoIP2DB(t *testing.T) {
	var err error

	// Create registration impl
	impl := &RegistrationImpl{params: &Params{}}

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
	err = impl.setNodeSequence(stateMap.GetNode(testID), stateMap.GetNode(testID).GetNodeAddresses())
	if err != nil {
		t.Errorf("setNodeSequence returned an error: %+v", err)
	}

	// Get node's new bin
	nodeDB, err := storage.PermissioningDb.GetNodeById(testID)
	if err != nil {
		t.Fatalf("Failed to get node: %+v", err)
	}

	if nodeDB.Sequence != "PH" {
		t.Errorf("setNodeSequence failed to set the expected bin."+
			"\nexpected: %s\nreceived: %s", "PH", nodeDB.Sequence)
	}

	ordering := stateMap.GetNode(testID).GetOrdering()
	if ordering != "PH" {
		t.Errorf("setNodeSequence failed to set the state ordering to the expected bin."+
			"\nexpected: %s\nreceived: %s", "PH", ordering)
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
	_ = impl.setNodeSequence(&node.State{}, "")
}
