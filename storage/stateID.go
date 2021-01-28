////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package storage defines the structure which creates and tracks the RoundID.
// It only allows itself to incremented forward by 1.
package storage

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/utils"
	"strconv"
	"strings"
	"sync"
)

// roundID structure contains the current ID and the file path to store it.
type stateID struct {
	id   uint64
	path string
	sync.RWMutex
}

// loadOrCreateStateID loads a new round ID from the specified file path and
// returns a new roundID with that ID. If no path is provided or the file does
// not exist, then the ID is set to startId.
func loadOrCreateStateID(path string, startId uint64) (*stateID, error) {
	// Skip reading from the file if no path is provided or file does not exist
	if path != "" && utils.FileExists(path) {
		roundIdBytes, err := utils.ReadFile(path)
		if err != nil {
			return nil, errors.Errorf("Could not load ID from file: %+v", err)
		}
		roundIdString := strings.TrimSpace(string(roundIdBytes))
		startId, err = strconv.ParseUint(roundIdString, 10, 64)
		if err != nil {
			return nil, errors.Errorf("Could not convert ID to uint: %+v", err)
		}
	} else {
		jww.WARN.Printf("Could not open state ID path %s because file does "+
			"not exist, reading ID from file skipped. state ID set to %d.",
			path, startId)
	}

	return &stateID{
		id:   startId,
		path: path,
	}, nil
}

// increment increments the ID by one, saves the new ID to file, and returns the
// previous ID. This function is thread safe. The internal value is updated only
// after the file write succeeds. If no path is provided, then only the ID in
// memory is updated.
func (rid *stateID) increment() (uint64, error) {
	rid.Lock()
	defer rid.Unlock()

	oldID := rid.id
	newID := rid.id + 1

	// Skip updating the file if no path is provided
	if rid.path != "" {
		// Convert the incremented ID to a string and write to file
		idBytes := []byte(strconv.FormatUint(newID, 10))
		err := utils.WriteFile(rid.path, idBytes, utils.FilePerms, utils.DirPerms)
		if err != nil {
			return 0, errors.Wrapf(err, "can't update to %d", newID)
		}
	} else {
		jww.WARN.Printf("The state ID path is empty, updating ID file skipped.")
	}

	// Update the ID in memory
	rid.id = newID

	return oldID, nil
}

// get returns the current ID.
func (rid *stateID) get() uint64 {
	rid.RLock()
	defer rid.RUnlock()

	return rid.id
}
