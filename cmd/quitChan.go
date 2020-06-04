////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

package cmd

import "sync"

// quitChan.go should be used to make it easier to end goroutines

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
