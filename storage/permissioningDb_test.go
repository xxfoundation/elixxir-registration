////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"errors"
	"fmt"
	"github.com/jinzhu/gorm"
	"gitlab.com/xx_network/primitives/id"
	"strconv"
	"testing"
	"time"
)

// Hidden function for one-time unit testing Database implementation
//func TestDatabaseImpl(t *testing.T) {
//	jww.SetLogThreshold(jww.LevelTrace)
//	jww.SetStdoutThreshold(jww.LevelTrace)
//
//	db, _, err := NewDatabase("cmix", "", "cmix_server", "0.0.0.0", "5432")
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//
//	err = db.InsertRoundMetric(&RoundMetric{
//		Id:            5,
//		PrecompStart:  time.Now(),
//		PrecompEnd:    time.Now(),
//		RealtimeStart: time.Now(),
//		RealtimeEnd:   time.Now().Add(-59*time.Minute),
//		BatchSize:     10,
//	}, nil)
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	err = db.InsertRoundMetric(&RoundMetric{
//		Id:            6,
//		PrecompStart:  time.Now(),
//		PrecompEnd:    time.Now(),
//		RealtimeStart: time.Now(),
//		RealtimeEnd:   time.Now().Add(-time.Hour),
//		BatchSize:     10,
//	}, nil)
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	err = db.InsertRoundMetric(&RoundMetric{
//		Id:            7,
//		PrecompStart:  time.Now(),
//		PrecompEnd:    time.Now(),
//		RealtimeStart: time.Now(),
//		RealtimeEnd:   time.Now().Add(-(3*time.Hour)),
//		BatchSize:     10,
//	}, nil)
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	result, result2, err := db.GetEarliestRound(2*time.Hour)
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	jww.FATAL.Printf("GetEarliestRound: %d %v", result, result2)
//
//	result, err = db.GetLatestEphemeralLength()
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//
//	err = db.InsertUser(&User{
//		PublicKey:             "test",
//		ReceptionKey:          "test",
//		RegistrationTimestamp: time.Now(),
//	})
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//	err = db.InsertUser(&User{
//		PublicKey:             "test",
//		ReceptionKey:          "test",
//		RegistrationTimestamp: time.Now(),
//	})
//	if err == nil {
//		t.Errorf("Expected duplicate key constraint")
//	}
//
//	jww.INFO.Printf("%+v", result)
//	result2, err := db.GetEphemeralLengths()
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//	jww.INFO.Printf("%#v", result2)
//
//	err = db.UpsertState(&State{
//		Key:   RoundIdKey,
//		Value: "10",
//	})
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//
//	val, err := db.GetStateValue(RoundIdKey)
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//	jww.FATAL.Printf(val)
//
//	err = db.UpsertState(&State{
//		Key:   RoundIdKey,
//		Value: "20",
//	})
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//
//	val, err = db.GetStateValue(RoundIdKey)
//	if err != nil {
//		t.Errorf(err.Error())
//	}
//	jww.FATAL.Printf(val)
//
//	testCode := "test"
//	testCode2 := "test2"
//	testId := id.NewIdFromString(testCode, id.Node, t)
//	testId2 := id.NewIdFromString(testCode2, id.Node, t)
//	testAppId := uint64(10010)
//	newApp := &Application{
//		Id:          testAppId,
//		Node:        Node{},
//		Name:        testCode,
//	}
//	newNode := &Node{
//		Code:               testCode,
//		Sequence:           testCode,
//		Status:             0,
//		ApplicationId:      testAppId,
//	}
//
//	err = db.InsertApplication(newApp, newNode)
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	err = db.RegisterNode(testId, nil,
//		testCode, "5.5.5.5", "test", "5.6.7.7", "test")
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//
//	err = db.UpdateLastActive([]*id.ID{testId, testId2})
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	err = db.UpdateNodeAddresses(testId, "6.6.6.6", "6.6.7.7")
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	nodes, err := db.GetNodes()
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	jww.FATAL.Printf("%+v", nodes[0])
//}

// Happy path
func TestDatabaseImpl_InsertNodeMetric(t *testing.T) {
	d, dc, err := NewDatabase("", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	db := d.GetDatabaseImpl(t)

	// Load in a registration code
	code := "TEST"
	testId := id.NewIdFromString(code, id.Node, t)
	applicationId := uint64(10)
	err = d.InsertApplication(&Application{Id: applicationId}, &Node{Code: code, Id: testId.Marshal()})
	if err != nil {
		t.Fatalf("Failed to set up reg code for registernode test: %+v", err)
	}

	newMetric := &NodeMetric{
		NodeId:    testId.Marshal(),
		StartTime: time.Now(),
		EndTime:   time.Now(),
		NumPings:  1000,
	}

	nodeMetricCounter := uint64(0)
	err = d.InsertNodeMetric(newMetric)
	if err != nil {
		t.Errorf("Unable to insert node metric: %+v", err)
	}
	nodeMetricCounter++

	var insertedMetric NodeMetric
	err = db.db.Take(&insertedMetric).Error
	if err != nil {
		t.Fatalf("Failed to get insertedmetric: %+v", err)
	}
	if insertedMetric.Id != nodeMetricCounter {
		t.Errorf("Mismatched ID returned!")
	}
	if insertedMetric.StartTime.UnixNano() != newMetric.StartTime.UnixNano() {
		t.Errorf("Mismatched StartTime returned!\n\tExpected: %s\n\tReceived: %s\n", newMetric.StartTime.String(), insertedMetric.StartTime.String())
	}
	if insertedMetric.EndTime.UnixNano() != newMetric.EndTime.UnixNano() {
		t.Errorf("Mismatched EndTime returned!\n\tExpected: %s\n\tReceived: %s\n", newMetric.EndTime.String(), insertedMetric.EndTime.String())
	}
	if insertedMetric.NumPings != newMetric.NumPings {
		t.Errorf("Mismatched NumPings returned!")
	}
}

// Happy path
func TestDatabaseImpl_InsertRoundMetric(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_InsertRoundMetric", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	db := d.GetDatabaseImpl(t)

	db.db.Create(&Topology{})

	roundId := uint64(1)
	newMetric := &RoundMetric{
		Id:            roundId,
		PrecompStart:  time.Now(),
		PrecompEnd:    time.Now(),
		RealtimeStart: time.Now(),
		RealtimeEnd:   time.Now(),
		RoundEnd:      time.Now(),
		BatchSize:     420,
	}
	newTopology := make([][]byte, 3)
	for i := 0; i < len(newTopology); i++ {
		nid := id.NewIdFromBytes([]byte(fmt.Sprintf("Node%d", i)), t)
		newTopology[i] = nid.Bytes()
		appId := uint64(i+1) * 10
		err = d.InsertApplication(&Application{Id: appId}, &Node{Code: fmt.Sprintf("TEST%d", i), Id: nid.Bytes()})
		if err != nil {
			t.Fatalf("Failed to insert node for test: %+v", err)
		}
	}

	err = d.InsertRoundMetric(newMetric, newTopology)
	if err != nil {
		t.Errorf("Unable to insert round metric: %+v", err)
	}

	var insertedMetric RoundMetric
	err = db.db.Take(&insertedMetric, "id = ?", roundId).Error
	if err != nil {
		t.Fatalf("Failed to get insertedmetric: %+v", err)
	}
	if insertedMetric.Id != roundId {
		t.Errorf("Mismatched ID returned!")
	}
	if insertedMetric.PrecompStart.UnixNano() != newMetric.PrecompStart.UnixNano() {
		t.Errorf("Mismatched PrecompStart returned!")
	}
	if insertedMetric.PrecompEnd.UnixNano() != newMetric.PrecompEnd.UnixNano() {
		t.Errorf("Mismatched PrecompEnd returned!")
	}
	if insertedMetric.RealtimeStart.UnixNano() != newMetric.RealtimeStart.UnixNano() {
		t.Errorf("Mismatched RealtimeStart returned!")
	}
	if insertedMetric.RealtimeEnd.UnixNano() != newMetric.RealtimeEnd.UnixNano() {
		t.Errorf("Mismatched RealtimeEnd returned!")
	}
	if insertedMetric.BatchSize != newMetric.BatchSize {
		t.Errorf("Mismatched BatchSize returned!")
	}
}

// Happy path
func TestDatabaseImpl_InsertRoundError(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_InsertRoundError", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	db := d.GetDatabaseImpl(t)

	roundId := id.Round(1)
	newMetric := &RoundMetric{
		Id:            uint64(roundId),
		PrecompStart:  time.Now(),
		PrecompEnd:    time.Now(),
		RealtimeStart: time.Now(),
		RealtimeEnd:   time.Now(),
		RoundEnd:      time.Now(),
		BatchSize:     420,
	}
	newTopology := make([][]byte, 3)
	for i := 0; i < len(newTopology); i++ {
		nid := id.NewIdFromBytes([]byte(fmt.Sprintf("Node%d", i)), t)
		newTopology[i] = nid.Bytes()
		appId := uint64(i+1) * 10
		err = d.InsertApplication(&Application{Id: appId}, &Node{Code: fmt.Sprintf("TEST%d", i), Id: nid.Bytes()})
		if err != nil {
			t.Fatalf("Failed to insert node for test: %+v", err)
		}
	}
	newErrors := []string{"err1", "err2", "err3"}

	err = d.InsertRoundMetric(newMetric, newTopology)
	if err != nil {
		t.Errorf("Unable to insert round metric: %+v", err)
	}

	err = d.InsertRoundError(roundId, newErrors[0])
	if err != nil {
		t.Errorf("Unable to insert round error: %+v", err)
	}

	err = d.InsertRoundError(roundId, newErrors[1])
	if err != nil {
		t.Errorf("Unable to insert round error: %+v", err)
	}

	var insertedMetric RoundMetric
	err = db.db.Preload("RoundErrors").Take(&insertedMetric, "id = ?", roundId).Error
	if err != nil {
		t.Fatalf("Failed to get insertedmetric: %+v", err)
	}

	if insertedMetric.RoundErrors[0].Error != newErrors[0] {
		t.Errorf("Mismatched Error returned!")
	}
	if insertedMetric.RoundErrors[1].Error != newErrors[1] {
		t.Errorf("Mismatched Error returned!")
	}
}

// Happy path
func TestDatabaseImpl_InsertEphemeralLength(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_InsertEphemeralLength", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	el := &EphemeralLength{
		Length:    10,
		Timestamp: time.Now(),
	}
	err = d.InsertEphemeralLength(el)
	if err != nil {
		t.Errorf("Unable to insert EphLen: %+v", err)
	}

	received, err := d.GetLatestEphemeralLength()
	if err != nil {
		t.Fatalf("Failed to get latest ephemeral length: %+v", err)
	}

	if received.Length != el.Length || received.Timestamp.UnixNano() != el.Timestamp.UnixNano() {
		t.Errorf("Expected to find inserted EphLen: %d", el.Length)
	}
}

// Error path
func TestDatabaseImpl_InsertEphemeralLengthErr(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_InsertEphemeralLengthErr", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	el := &EphemeralLength{
		Length:    10,
		Timestamp: time.Now(),
	}
	// Manually add duplicate entry
	err = d.InsertEphemeralLength(el)
	if err != nil {
		t.Fatalf("Failed to add initial entry for duplicate test: %+v", err)
	}

	err = d.InsertEphemeralLength(el)
	if err == nil {
		t.Errorf("Expected failure from duplicate EphLen!")
	}
}

// Happy path
func TestDatabaseImpl_GetEphemeralLengths(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetEphemeralLengths", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	testLen := 64

	// Make a bunch of results to insert
	for i := 1; i <= testLen; i++ {
		el := &EphemeralLength{
			Length:    uint8(i),
			Timestamp: time.Now(),
		}
		err = d.InsertEphemeralLength(el)
		if err != nil {
			t.Fatalf("Failed to insert ephemeral len %d: %+v", i, err)
		}
	}

	result, err := d.GetEphemeralLengths()
	if err != nil {
		t.Errorf("Unable to get all EphLen: %+v", err)
	}

	if len(result) != testLen {
		t.Errorf("Didn't get correct number of EphLen, Got %d Expected %d", len(result), testLen)
	}
}

// Error path
func TestDatabaseImpl_GetEphemeralLengthsErr(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetEphemeralLengthsErr", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	result, err := d.GetEphemeralLengths()
	if err != nil {
		t.Fatalf("GetEphLens won't return error, just nil when bad: %+v", err)
	}
	if len(result) != 0 {
		t.Errorf("Should not receive any ephlens: %+v", result)
	}
}

// Happy path
func TestDatabaseImpl_GetLatestEphemeralLength(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetLatestEphemeralLength", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	// Make a bunch of results to insert
	maxLen := 50
	for i := 0; i <= maxLen; i += 5 {

		el := &EphemeralLength{
			Length: uint8(i + 1),
			// Unlike the real world, decrease Timestamp as Length increases
			// in order to ensure latest EphemeralLength is based on Length
			Timestamp: time.Now().Add(time.Duration(-i) * time.Minute),
		}
		err = d.InsertEphemeralLength(el)
		if err != nil {
			t.Fatalf("Failed to insert ephemeral length %d (%d): %+v", i, el.Length, err)
		}
	}

	result, err := d.GetLatestEphemeralLength()
	if err != nil {
		t.Errorf("Unable to get latest EphLen: %+v", err)
	}

	if result.Length != uint8(maxLen+1) {
		t.Errorf("Latest EphLen incorrect: Got %d, expected %d", result.Length, maxLen)
	}
}

// Error path
func TestDatabaseImpl_GetLatestEphemeralLengthErr(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetLatestEphemeralLengthErr", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	_, err = d.GetLatestEphemeralLength()
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("Expected record not found err, instead got %+v", err)
	}
}

func TestDatabaseImpl_GetEarliestRound(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetEarliestRound", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	cutoff := 20 * time.Minute
	roundId, _, err := d.GetEarliestRound(cutoff)
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) || int(roundId) != 0 {
		t.Errorf("Invalid return for empty roundMetrics: (%d) %+v", roundId, err)
	}

	metrics := []*RoundMetric{{
		Id:            1,
		PrecompStart:  time.Now(),
		PrecompEnd:    time.Now(),
		RealtimeStart: time.Now(),
		RealtimeEnd:   time.Now().Add(-30 * time.Minute),
		RoundEnd:      time.Now(),
		BatchSize:     420,
	},
		{
			Id:            2,
			PrecompStart:  time.Now(),
			PrecompEnd:    time.Now(),
			RealtimeStart: time.Now(),
			RealtimeEnd:   time.Now().Add(-time.Minute),
			RoundEnd:      time.Now(),
			BatchSize:     420,
		},
		{
			Id:            3,
			PrecompStart:  time.Now(),
			PrecompEnd:    time.Now(),
			RealtimeStart: time.Now(),
			RealtimeEnd:   time.Now().Add(-10 * time.Minute),
			RoundEnd:      time.Now(),
			BatchSize:     420,
		},
	}
	newTopology := make([][]byte, 1)
	for i := 0; i < len(newTopology); i++ {
		nid := id.NewIdFromBytes([]byte(fmt.Sprintf("Node%d", i)), t)
		newTopology[i] = nid.Bytes()
		appId := uint64(i+1) * 10
		err = d.InsertApplication(&Application{Id: appId}, &Node{Code: fmt.Sprintf("TEST%d", i), Id: nid.Bytes()})
		if err != nil {
			t.Fatalf("Failed to insert node for test: %+v", err)
		}
	}

	// Insert dummy metrics
	for _, metric := range metrics {
		err = d.InsertRoundMetric(metric, newTopology)
		if err != nil {
			t.Fatalf("Failed to insert round metric: %+v", err)
		}
	}

	roundId, _, err = d.GetEarliestRound(cutoff)
	if err != nil || uint64(roundId) != 3 {
		t.Errorf("Invalid return for GetEarliestRound: %d %+v", roundId, err)
	}
}

// Test error path to ensure error message stays consistent
func TestDatabaseImpl_GetStateValue(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetStateValue", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()

	_, err = d.GetStateValue("test")
	if err == nil {
		t.Errorf("Expected error getting bad state value!")
		return
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("Invalid error message getting bad state value: Got %s", err.Error())
	}
}

// Unit test
func TestDatabaseImpl_GetBin(t *testing.T) {
	d, dc, err := NewDatabase("", "", "TestDatabaseImpl_GetBin", "", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		err := dc()
		if err != nil {
			t.Errorf("Failed to close database: %+v", err)
		}
	}()
	db := d.GetDatabaseImpl(t)

	binList, err := d.getBins()
	if err != nil {
		t.Errorf("getBins will not error with no bins found")
	}
	if len(binList) > 0 {
		t.Fatalf("Expected empty list, instead got %+v", binList)
	}

	// Set up and populate the map with testing values
	testStrings := []string{"0", "1", "2", "3", "4"}
	expectedMap := make(map[GeoBin]struct{})
	for _, code := range testStrings {
		bin, err := strconv.Atoi(code)
		if err != nil {
			t.Fatalf("Failed on setup: %v", err)
		}
		err = db.db.Create(&GeoBin{
			Country: code,
			Bin:     uint8(bin),
		}).Error
		if err != nil {
			t.Fatalf("Failed to create bin: %+v", err)
		}
		expectedBin := &GeoBin{Bin: uint8(bin), Country: code}
		expectedMap[*expectedBin] = struct{}{}
	}

	// Pull the bins
	receivedBins, err := d.getBins()
	if err != nil {
		t.Fatalf("Unexpcted error in getBins(): %v", err)
	}

	// Test the results
	if len(receivedBins) != len(expectedMap) {
		t.Errorf("Did not receive expected amount of bins."+
			"\n\tExpected: %d"+
			"\n\tReceived: %v", len(expectedMap), len(receivedBins))
	}

	for _, bin := range receivedBins {
		_, ok := expectedMap[*bin]
		if !ok {
			t.Errorf("Retrieved unexpected bin from map.")
		}
	}

}
