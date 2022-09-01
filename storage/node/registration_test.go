////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package node

import (
	"gitlab.com/xx_network/primitives/utils"
	"os"
	"reflect"
	"testing"
)

// Tests that LoadInfo() correctly reads and unmarshals data from a file.
func TestLoadInfo(t *testing.T) {
	// Set up expected values
	filePath := "testRegCodes.json"
	testData := []byte("[{\"RegCode\":\"GMWJGSAPGA\",\"Order\":\"TLMODYLUMG\"" +
		"},{\"RegCode\":\"UQDIWWMNPP\",\"Order\":\"QIMGEGMGKB\"},{\"RegCode\"" +
		":\"NPTNIWDYJD\",\"Order\":\"HVOYAFNNDF\"},{\"RegCode\":\"BKFGQDLCIV" +
		"\",\"Order\":\"NBCSWTCNVU\"},{\"RegCode\":\"XDMIQJISQC\",\"Order\":" +
		"\"DLVHPDCSFX\"},{\"RegCode\":\"ABPHMKKKPH\",\"Order\":\"XDQHJWRMVT\"" +
		"}]")
	testInfos := []Info{
		{RegCode: "GMWJGSAPGA", Order: "TLMODYLUMG"},
		{RegCode: "UQDIWWMNPP", Order: "QIMGEGMGKB"},
		{RegCode: "NPTNIWDYJD", Order: "HVOYAFNNDF"},
		{RegCode: "BKFGQDLCIV", Order: "NBCSWTCNVU"},
		{RegCode: "XDMIQJISQC", Order: "DLVHPDCSFX"},
		{RegCode: "ABPHMKKKPH", Order: "XDQHJWRMVT"},
	}

	// Defer deletion of the test JSON file
	defer func() {
		// Clean up temporary test file
		err := os.RemoveAll(filePath)
		if err != nil {
			t.Fatalf("Error deleting test JSON file %s:\n\t%v", filePath, err)
		}
	}()

	// Create test JSON file
	err := utils.WriteFile(filePath, testData, utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Error creating test JSON file %s:\n\t%v", filePath, err)
	}

	// Call LoadInfo()
	infos, err := LoadInfo(filePath)
	if err != nil {
		t.Errorf("LoadInfo() encountered an error loading the JSON from"+
			" %s:\n\t%v", filePath, err)
	}

	// Compare received Info slice to expected
	if !reflect.DeepEqual(infos, testInfos) {
		t.Errorf("LoadInfo() marshaled the JSON data incorrectly."+
			"\n\texpected: %+v\n\treceived: %+v", testInfos, infos)
	}
}

// Tests that LoadInfo produces an error when given an invalid file path.
func TestLoadInfo_FileError(t *testing.T) {
	// Set up expected values
	filePath := "testRegCodes.json"
	var testInfos []Info
	expectedErr := "Could not load JSON file: open testRegCodes.json: The " +
		"system cannot find the file specified."

	infos, err := LoadInfo(filePath)
	if err == nil {
		t.Errorf("LoadInfo() did not encounter an error when it should "+
			"have while loading the JSON from %s\n\texpected: %s\n\treceived: %v",
			filePath, expectedErr, err)
	}

	if !reflect.DeepEqual(infos, testInfos) {
		t.Errorf("LoadInfo() should have returned empty []Info."+
			"\n\texpected: %+v\n\treceived: %+v", testInfos, infos)
	}
}

// Tests that LoadInfo() produces an error when given invalid JSON.
func TestLoadInfo_JsonError(t *testing.T) {
	// Set up expected values
	filePath := "testRegCodes.json"
	testData := []byte("This is invalid JSON.")
	expectedErr := "Could not unmarshal JSON: invalid character 'T' looking " +
		"for beginning of value"
	var testInfos []Info

	// Defer deletion of the test JSON file
	defer func() {
		// Clean up temporary test file
		err := os.RemoveAll(filePath)
		if err != nil {
			t.Fatalf("Error deleting test JSON file %s:\n\t%v", filePath, err)
		}
	}()

	// Create test JSON file
	err := utils.WriteFile(filePath, testData, utils.FilePerms, utils.DirPerms)
	if err != nil {
		t.Fatalf("Error creating test JSON file %s:\n\t%v", filePath, err)
	}

	// Call LoadInfo()
	infos, err := LoadInfo(filePath)
	if err == nil {
		t.Errorf("LoadInfo() did not encounter an error when it should "+
			"have while unmarshaling the JSON from %s\n\texpected: %s\n\treceived: %v",
			filePath, expectedErr, err)
	}

	// Compare received Info slice to expected
	if !reflect.DeepEqual(infos, testInfos) {
		t.Errorf("LoadInfo() should have returned empty []Info."+
			"\n\texpected: %+v\n\treceived: %+v", testInfos, infos)
	}
}
