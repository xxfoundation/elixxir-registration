////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// +build !windows

package cmd

import (
	"syscall"
	"testing"
	"time"
)

func TestReceiveSIGUSR1Signal(t *testing.T) {
	called := make(chan bool, 1)
	testfn := func() {
		called <- true
	}

	go ReceiveUSR1Signal(testfn)
	// Give a little bit of time for the subthread to start picking up
	// the signal
	time.Sleep(100 * time.Millisecond)

	syscall.Kill(syscall.Getpid(), syscall.SIGUSR1)

	res := false
	// Sleep multiple times to give the kernel more tries to
	// deliver the signal.
	for i := 0; i < 10; i++ {
		select {
		case res = <-called:
			break
		case <-time.After(100 * time.Millisecond):
		}
	}

	if res != true {
		t.Errorf("Signal USR1 was not handled!")
	}
}

func TestReceiveSIGUSR2Signal(t *testing.T) {
	called := make(chan bool, 1)
	testfn := func() {
		called <- true
	}

	go ReceiveUSR2Signal(testfn)
	// Give a little bit of time for the subthread to start picking up
	// the signal
	time.Sleep(100 * time.Millisecond)

	syscall.Kill(syscall.Getpid(), syscall.SIGUSR2)

	res := false
	// Sleep multiple times to give the kernel more tries to
	// deliver the signal.
	for i := 0; i < 10; i++ {
		select {
		case res = <-called:
			break
		case <-time.After(100 * time.Millisecond):
		}
	}

	if res != true {
		t.Errorf("Signal USR1 was not handled!")
	}
}

//func TestReceiveExitSignal(t *testing.T) {
//	called := make(chan bool, 1)
//	testfn := func() int {
//		called <- true
//		return 0
//	}
//
//	go ReceiveExitSignal(testfn)
//	time.Sleep(1 * time.Second)
//
//	syscall.Kill(syscall.Getpid(), syscall.SIGINT)
//
//	// Sleep multiple times to give the kernel more tries to
//	// deliver the signal.
//	res := false
//	for i := 0; i < 20; i++ {
//		select {
//		case res = <-called:
//			break
//		case <-time.After(150 * time.Millisecond):
//		}
//	}
//
//	if res != true {
//		t.Errorf("Signal INT was not handled!")
//	}
//}
