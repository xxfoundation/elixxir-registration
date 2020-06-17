package cmd

import (
	"github.com/pkg/errors"
	"strings"
	"testing"
)

func TestRecoverable(t *testing.T) {
	panicfunc := func() error {
		panic("Failed")
	}
	normfunc := func() error {
		return errors.New("Error message two")
	}

	err1 := recoverable(panicfunc, "test str one")

	if err1 == nil {
		t.Error("Recovery did not succeed")
	}
	if !strings.Contains(err1.Error(), "test str one") {
		t.Error("Did not return proper error")
	}

	err2 := recoverable(normfunc, "test str")

	if err2 == nil {
		t.Error("Recovery did not succeed second time")
	}
	if !strings.Contains(err2.Error(), "Error message two") {
		t.Error("Did not receive correct error message")
	}
}
