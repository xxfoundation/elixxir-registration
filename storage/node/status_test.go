package node

import "testing"

//tests that the stringer of Status are correct
func TestStatus_String(t *testing.T) {

	expected := []string{"Unregistered", "Active", "Inactive", "Banned",
		"Unknown"}

	for i := 0; i < 5; i++ {
		s := Status(i)
		if s.String() != expected[i] {
			t.Errorf("Stringer of status %v incoorect; "+
				"expected: %s, recieved: %s", i, expected[i], s.String())
		}

	}
}
