////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package storage

import (
	"gitlab.com/xx_network/primitives/id"
	"testing"
	"time"
)

// Hidden function for one-time unit testing Database implementation
//func TestDatabaseImpl(t *testing.T) {
//	db, _, err := NewDatabase("cmix", "", "cmix_server", "0.0.0.0", "5432")
//	if err != nil {
//		t.Errorf(err.Error())
//		return
//	}
//
//	testCode := "test"
//	testId := id.NewIdFromString(testCode, id.Node, t)
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
