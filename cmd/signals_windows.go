////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

//go:build windows
// +build windows

// signals.go handles signals specific to the permissioning server:
//   - SIGUSR1, which stops round creation
//   - SIGTERM/SIGINT, which stops round creation and exits
//
// The functions are set up to receive arbitrary functions that handle
// the necessary behaviors.

package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"os/signal"
	"syscall"
)

// ReceiveSignal is a dummy function because windows doesn't have the support
// we need.
func ReceiveSignal(sigFn func(), sig os.Signal) {
	jww.WARN.Printf("Windows does not support all signals, ignored!")
}

// ReceiveUSR1Signal is a dummy function because windows doesn't have the
// support we need.
func ReceiveUSR1Signal(usr1Fn func()) {
	jww.WARN.Printf("Windows does not support SIGUSR1 signals, ignored!")
}

// ReceiveUSR2Signal is a dummy function because windows doesn't have the
// support we need.
func ReceiveUSR2Signal(usr1Fn func()) {
	jww.WARN.Printf("Windows does not support SIGUSR2 signals, ignored!")
}

// ReceiveExitSignal calls the provided exit function and exits
// with the provided exit status when the program receives
// SIGTERM or SIGINT
func ReceiveExitSignal() chan os.Signal {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	return c
}
