////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// +build !windows

// signals.go handles signals specific to the permissioning server:
//   - SIGUSR1, which stops round creation
//   - SIGTERM/SIGINT, which stops round creation and exits
//
// The functions are set up to receive arbitrary functions that handle
// the necessary behaviors instead of implementing the behavior directly.

package cmd

import (
	jww "github.com/spf13/jwalterweatherman"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// ReceiveSignal calls the provided function when it receives a specific
// signal. It will call the provided function every time it recieves the signal.
func ReceiveSignal(sigFn func(), sig os.Signal) {
	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	c := make(chan os.Signal, 1)
	signal.Notify(c, sig)

	// Block until a signal is received, then call the function
	// provided
	for {
		<-c
		jww.INFO.Printf("Received %s signal...\n", sig)
		sigFn()
	}
}

// ReceiveUSR1Signal calls the provided function when receiving SIGUSR1.
// It will call the provided function every time it receives it
func ReceiveUSR1Signal(usr1Fn func()) {
	ReceiveSignal(usr1Fn, syscall.SIGUSR1)
}

// ReceiveUSR2Signal calls the provided function when receiving SIGUSR1.
// It will call the provided function every time it receives it
func ReceiveUSR2Signal(usr1Fn func()) {
	ReceiveSignal(usr1Fn, syscall.SIGUSR2)
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
	jww.INFO.Printf("Received Exit (SIGTERM or SIGINT) signal...\n")
	ret := exitFn()
	os.Exit(ret)
}

type QuitChan chan struct{}

type QuitChans struct {
	quitChans     []QuitChan
	quitChansLock sync.Mutex
}

// Makes and registers a simple quit channel that will get notified on sigusr2
func (qcs *QuitChans) MakeQuitChan() QuitChan {
	qcs.quitChansLock.Lock()
	defer qcs.quitChansLock.Unlock()

	// Make a channel suitable for one non-blocking send
	quitChan := make(QuitChan)
	qcs.quitChans = append(qcs.quitChans, quitChan)
	return quitChan
}
