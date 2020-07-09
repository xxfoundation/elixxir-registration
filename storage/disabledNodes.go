////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the loading of disabled Nodes from a text file.

package storage

import (
	"encoding/base64"
	"fmt"
	"github.com/golang-collections/collections/set"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/storage/node"
	"strings"
	"sync"
	"time"
)

// disabledNodes contains a set of Node states that should be disabled from
// running rounds. It is updated in its own thread from a text file. The mutex
// prevents updating the the list while it is being read.
type disabledNodes struct {
	nodes    *set.Set      // List of disabled nodes
	path     string        // Path to list of disabled Nodes
	interval time.Duration // Interval between polls to update list
	sync.RWMutex
}

// generateDisabledNodes reads the file at the path and generates a new
// disabledNodes with the contents. If the file cannot be read, than an error
// is returned. If the file can be read but the IDs cannot be parsed, then a
// warning is printed and the new object is returned.
func generateDisabledNodes(path string, interval time.Duration,
	state *NetworkState) (*disabledNodes, error) {
	// Get file contents
	fileBytes, err := utils.ReadFile(path)
	if err != nil {
		return nil, errors.Errorf("Skipping polling of disabled node ID list "+
			"file; error while accessing file: %v", err)
	}

	// Parse the file contents into a set of node states with the disabled IDs
	nodeSet, err := getDisabledNodesSet(string(fileBytes), state.GetNodeMap())
	if err != nil {
		jww.WARN.Printf("Error while parsing disabled Node list: %v", err)
	}

	// Create new disabledNodes object
	dnl := &disabledNodes{
		nodes:    nodeSet,
		path:     path,
		interval: interval,
	}

	return dnl, nil
}

// pollDisabledNodes initialises a disabled Node list from the specified file
// and starts a thread that updates the list from the file at the specified
// interval. The provided channel allows for external killing of the routine.
func (dnl *disabledNodes) pollDisabledNodes(state *NetworkState, quitChan chan struct{}) {
	ticker := time.NewTicker(dnl.interval)
	jww.DEBUG.Printf("Starting disabled Node list updater thread polling "+
		"every %s", dnl.interval.String())

	for {
		select {
		case <-quitChan:
			jww.DEBUG.Printf("Killing disabled Nodes polling routine.")
			return
		case <-ticker.C:
			// Get file contents and skip parsing contents on error
			fileBytes, err := utils.ReadFile(dnl.path)
			if err != nil {
				jww.WARN.Printf("Error while accessing disbaled Node list "+
					"file: %v", err)
				continue
			}

			// Parse the file contents into a set of node states with the
			// disabled IDs
			nodeSet, err := getDisabledNodesSet(string(fileBytes),
				state.GetNodeMap())
			if err != nil {
				jww.WARN.Printf("Error while parsing disbaled Node list: "+
					"%v", err)
			}

			// Update the disabled Nodes list (thread safe)
			dnl.updateDisabledNodes(nodeSet)
		}
	}
}

// updateDisabledNodes copies the values from the new Node set into the
// disabled Node list. This function is thread safe.
func (dnl *disabledNodes) updateDisabledNodes(newSet *set.Set) {
	dnl.Lock()
	dnl.nodes = newSet
	dnl.Unlock()
}

// getDisabledNodes returns a copy of the list of Node States of Node that
// should be excluded from team forming. This function is thread safe.
func (dnl *disabledNodes) getDisabledNodes() *set.Set {
	dnl.RLock()
	defer dnl.RUnlock()
	return dnl.nodes
}

// getDisabledNodesSet parses the delineated Node ID string into a Set of Node
// states. Any ID strings that fail to be base64 decoded, unmarshalled, or found
// in the StateMap are skipped and an error is recorded. All errors are returned
// at the end in a group. A text file with Node IDs in base64 string format
// separated by new lines (\n) is expected.
func getDisabledNodesSet(idList string, states *node.StateMap) (*set.Set, error) {

	// Trim whitespace from the start and end of the file contents
	nodeListString := strings.TrimSpace(idList)

	// Convert \n separated ID strings to an array
	nodeListArr := strings.Split(nodeListString, "\n")

	stateList := set.New()
	var errs string
	var combinedErrors error

	// Loop through each string, convert it, and store its state, if it exists
	for i, idString := range nodeListArr {
		// Trim extra spaces at beginning and end of ID, which allows it to
		// support extraneous space such as in Windows line breaks (\r\n)
		idString = strings.TrimSpace(idString)

		// Decode base64 ID to bytes
		nodeID, err1 := base64.StdEncoding.DecodeString(idString)
		if err1 != nil {
			errs += fmt.Sprintf("\tFailed to base64 decode ID %s at index %d: "+
				"%v\n", idString, i, err1)
			continue
		}

		// Convert ID bytes to an ID
		newId, err1 := id.Unmarshal(nodeID)
		if err1 != nil {
			errs += fmt.Sprintf("\tFailed to unmarshal ID %#v at index %d: "+
				"%v\n", idString, i, err1)
			continue
		}

		// Add Node state with the ID to the Set, if it exists
		newState := states.GetNode(newId)
		if newState != nil {
			stateList.Insert(newState)
		} else {
			errs += fmt.Sprintf("\tFailed to find ID %#v in state map at index "+
				"%d.\n", idString, i)
		}
	}

	// If any error messages were recorded, convert them into an error
	if errs != "" {
		combinedErrors = errors.Errorf("Encountered issue(s) parsing IDs:\n%s",
			errs)
	}

	// Return a nil Set if no states were added
	if stateList.Len() < 1 {
		stateList = nil
	}

	return stateList, combinedErrors
}
