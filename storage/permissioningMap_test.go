////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"gitlab.com/xx_network/primitives/id"
	"strings"
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
//	result, err := db.GetLatestEphemeralLength()
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
//	testId := id.NewIdFromString(testCode, id.Node, t)
//	testAppId := uint64(10010)
//	newApp := &Application{
//		Id:   testAppId,
//		Node: Node{},
//		Name: testCode,
//	}
//	newNode := &Node{
//		Code:          testCode,
//		Sequence:      testCode,
//		Status:        0,
//		ApplicationId: testAppId,
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
//	err = db.UpdateNodeAddresses(testId, "6.6.6.6", "6.6.7.7")
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//	err = db.UpdateSalt(testId, []byte("test123"))
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//}

// Happy path
func TestMapImpl_InsertNodeMetric(t *testing.T) {
	m := &MapImpl{nodeMetrics: make(map[uint64]*NodeMetric)}

	newMetric := &NodeMetric{
		NodeId:    []byte("TEST"),
		StartTime: time.Now(),
		EndTime:   time.Now(),
		NumPings:  1000,
	}

	err := m.InsertNodeMetric(newMetric)
	if err != nil {
		t.Errorf("Unable to insert node metric: %+v", err)
	}

	insertedMetric := m.nodeMetrics[m.nodeMetricCounter]
	if insertedMetric.Id != m.nodeMetricCounter {
		t.Errorf("Mismatched ID returned!")
	}
	if insertedMetric.StartTime != newMetric.StartTime {
		t.Errorf("Mismatched StartTime returned!")
	}
	if insertedMetric.EndTime != newMetric.EndTime {
		t.Errorf("Mismatched EndTime returned!")
	}
	if insertedMetric.NumPings != newMetric.NumPings {
		t.Errorf("Mismatched NumPings returned!")
	}
}

// Happy path
func TestMapImpl_InsertRoundMetric(t *testing.T) {
	m := &MapImpl{roundMetrics: make(map[uint64]*RoundMetric)}

	roundId := uint64(1)
	newMetric := &RoundMetric{
		Id:            roundId,
		PrecompStart:  time.Now(),
		PrecompEnd:    time.Now(),
		RealtimeStart: time.Now(),
		RealtimeEnd:   time.Now(),
		BatchSize:     420,
	}
	newTopology := [][]byte{id.NewIdFromBytes([]byte("Node1"), t).Bytes(),
		id.NewIdFromBytes([]byte("Node2"), t).Bytes()}

	err := m.InsertRoundMetric(newMetric, newTopology)
	if err != nil {
		t.Errorf("Unable to insert round metric: %+v", err)
	}

	insertedMetric := m.roundMetrics[roundId]
	if insertedMetric.Id != roundId {
		t.Errorf("Mismatched ID returned!")
	}
	if insertedMetric.PrecompStart != newMetric.PrecompStart {
		t.Errorf("Mismatched PrecompStart returned!")
	}
	if insertedMetric.PrecompEnd != newMetric.PrecompEnd {
		t.Errorf("Mismatched PrecompEnd returned!")
	}
	if insertedMetric.RealtimeStart != newMetric.RealtimeStart {
		t.Errorf("Mismatched RealtimeStart returned!")
	}
	if insertedMetric.RealtimeEnd != newMetric.RealtimeEnd {
		t.Errorf("Mismatched RealtimeEnd returned!")
	}
	if insertedMetric.BatchSize != newMetric.BatchSize {
		t.Errorf("Mismatched BatchSize returned!")
	}
}

// Happy path
func TestMapImpl_InsertRoundError(t *testing.T) {
	m := &MapImpl{roundMetrics: make(map[uint64]*RoundMetric)}

	roundId := id.Round(1)
	newMetric := &RoundMetric{
		Id:            uint64(roundId),
		PrecompStart:  time.Now(),
		PrecompEnd:    time.Now(),
		RealtimeStart: time.Now(),
		RealtimeEnd:   time.Now(),
		BatchSize:     420,
	}
	newTopology := [][]byte{id.NewIdFromBytes([]byte("Node1"), t).Bytes(),
		id.NewIdFromBytes([]byte("Node2"), t).Bytes()}
	newErrors := []string{"err1", "err2"}

	err := m.InsertRoundMetric(newMetric, newTopology)
	if err != nil {
		t.Errorf("Unable to insert round metric: %+v", err)
	}

	insertedMetric := m.roundMetrics[uint64(roundId)]

	err = m.InsertRoundError(roundId, newErrors[0])
	if err != nil {
		t.Errorf("Unable to insert round error: %+v", err)
	}

	err = m.InsertRoundError(roundId, newErrors[1])
	if err != nil {
		t.Errorf("Unable to insert round error: %+v", err)
	}

	if insertedMetric.RoundErrors[0].Error != newErrors[0] {
		t.Errorf("Mismatched Error returned!")
	}
	if insertedMetric.RoundErrors[1].Error != newErrors[1] {
		t.Errorf("Mismatched Error returned!")
	}
}

// Happy path
func TestMapImpl_InsertEphemeralLength(t *testing.T) {
	m := &MapImpl{ephemeralLengths: make(map[uint8]*EphemeralLength)}

	el := &EphemeralLength{
		Length:    10,
		Timestamp: time.Now(),
	}
	err := m.InsertEphemeralLength(el)
	if err != nil {
		t.Errorf("Unable to insert EphLen: %+v", err)
	}

	if m.ephemeralLengths[el.Length] == nil {
		t.Errorf("Expected to find inserted EphLen: %d", el.Length)
	}
}

// Error path
func TestMapImpl_InsertEphemeralLengthErr(t *testing.T) {
	m := &MapImpl{ephemeralLengths: make(map[uint8]*EphemeralLength)}

	el := &EphemeralLength{
		Length:    10,
		Timestamp: time.Now(),
	}
	// Manually add duplicate entry
	m.ephemeralLengths[el.Length] = el

	err := m.InsertEphemeralLength(el)
	if err == nil {
		t.Errorf("Expected failure from duplicate EphLen!")
	}
}

// Happy path
func TestMapImpl_GetEphemeralLengths(t *testing.T) {
	m := &MapImpl{ephemeralLengths: make(map[uint8]*EphemeralLength)}
	testLen := 64

	// Make a bunch of results to insert
	for i := 0; i < testLen; i++ {
		el := &EphemeralLength{
			Length:    uint8(i),
			Timestamp: time.Now(),
		}
		m.ephemeralLengths[el.Length] = el
	}

	result, err := m.GetEphemeralLengths()
	if err != nil {
		t.Errorf("Unable to get all EphLen: %+v", err)
	}

	if len(result) != testLen {
		t.Errorf("Didn't get correct number of EphLen, Got %d Expected %d", len(result), testLen)
	}
}

// Error path
func TestMapImpl_GetEphemeralLengthsErr(t *testing.T) {
	m := &MapImpl{ephemeralLengths: make(map[uint8]*EphemeralLength)}
	result, err := m.GetEphemeralLengths()
	if result != nil || err == nil {
		t.Errorf("Expected error getting bad EphLens!")
	}
}

// Happy path
func TestMapImpl_GetLatestEphemeralLength(t *testing.T) {
	m := &MapImpl{ephemeralLengths: make(map[uint8]*EphemeralLength)}

	// Make a bunch of results to insert
	maxLen := 50
	for i := 0; i <= maxLen; i += 5 {

		el := &EphemeralLength{
			Length: uint8(i),
			// Unlike the real world, decrease Timestamp as Length increases
			// in order to ensure latest EphemeralLength is based on Length
			Timestamp: time.Now().Add(time.Duration(-i) * time.Minute),
		}
		m.ephemeralLengths[el.Length] = el
	}

	result, err := m.GetLatestEphemeralLength()
	if err != nil {
		t.Errorf("Unable to get latest EphLen: %+v", err)
	}

	if result.Length != uint8(maxLen) {
		t.Errorf("Latest EphLen incorrect: Got %d, expected %d", result.Length, maxLen)
	}
}

// Error path
func TestMapImpl_GetLatestEphemeralLengthErr(t *testing.T) {
	m := &MapImpl{ephemeralLengths: make(map[uint8]*EphemeralLength)}
	result, err := m.GetLatestEphemeralLength()
	if result != nil || err == nil {
		t.Errorf("Expected error getting bad latest EphLen!")
	}
}

// Test error path to ensure error message stays consistent
func TestMapImpl_GetStateValue(t *testing.T) {
	m := &MapImpl{states: make(map[string]string)}

	_, err := m.GetStateValue("test")
	if err == nil {
		t.Errorf("Expected error getting bad state value!")
		return
	}

	if !strings.Contains(err.Error(), "Unable to locate state for key") {
		t.Errorf("Invalid error message getting bad state value: Got %s", err.Error())
	}
}
