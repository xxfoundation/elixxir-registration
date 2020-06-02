////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

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

// ReceiveUSR1Signal calls the provided function when receiving SIGUSR1.
// It will call the provided function every time it receives it
func ReceiveUSR1Signal(usr1Fn func()) {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGUSR1)

	// Block until a signal is received, then call the function
	// provided
	for {
		<-c
		jww.INFO.Printf("Received SIGUSR1 signal...")
		usr1Fn()
	}
}

// ReceiveExitSignal calls the provided exit function and exits
// with the provided exit status when the program receives
// SIGTERM or SIGINT
func ReceiveExitSignal(exitFn func() int) {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received, then call the function
	// provided
	<-c
	jww.INFO.Printf("Received Exit (SIGTERM or SIGINT) signal...")
	ret := exitFn()
	os.Exit(ret)
}
