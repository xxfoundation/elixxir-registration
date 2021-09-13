////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/xx_network/primitives/id"
	"time"
)

// Error messages.
const (
	getActiveNodesDbErr      = "failed to get active node list from database: %+v"
	unmarshalActiveNodeDbErr = "failed to unmarshal active node ID #%d: %+v"
)

func TrackNodeMetrics(impl *RegistrationImpl, quitChan chan struct{},
	nodeMetricInterval time.Duration) {
	jww.DEBUG.Printf("Beginning storage of node metrics every %+v...",
		nodeMetricInterval)
	nodeTicker := time.NewTicker(nodeMetricInterval)
	onlyScheduleActive := impl.params.onlyScheduleActive
	for {
		// Store the metric start time
		startTime := time.Now()
		select {
		case <-quitChan:
			return
		// Wait for the ticker to fire
		case <-nodeTicker.C:
			var toPrune []*id.ID
			var active map[id.ID]bool
			var err error
			if onlyScheduleActive {
				active, err = GetActiveNodeIDs()
				if err != nil {
					jww.ERROR.Print(err)
				}
				jww.DEBUG.Printf("Found %d active nodes!", len(active))

				// Serialize the active node map
				activeNodes := make([]*id.ID, 0, len(active))
				for activeId := range active {
					activeNodes = append(activeNodes, &activeId)
				}

				// Update all the active nodes in the database
				err = storage.PermissioningDb.UpdateLastActive(activeNodes)
				if err != nil {
					jww.ERROR.Printf("TrackNodeMetrics: Could not update last active: %v", err)
				}
			}

			nodes, err := storage.PermissioningDb.GetNodes()
			if err != nil {
				jww.ERROR.Printf("TrackNodeMetrics: Could not retrieve nodes: %+v", err)
			}

			// Place in a map
			dbMap := make(map[id.ID]*storage.Node)
			for _, node := range nodes {
				nid, err := id.Unmarshal(node.Id)
				if err != nil {
					jww.ERROR.Printf("Could not unmarshal ID from database: %+v", err)
					continue
				}

				dbMap[*nid] = node
			}

			// Iterate over the Node States
			nodeStates := impl.State.GetNodeMap().GetNodeStates()
			var toUpdate []*id.ID
			for _, nodeState := range nodeStates {

				// Build the NodeMetric
				currentTime := time.Now()
				metric := &storage.NodeMetric{
					NodeId:    nodeState.GetID().Bytes(),
					StartTime: startTime,
					EndTime:   currentTime,
					NumPings:  nodeState.GetAndResetNumPolls(),
				}

				// set the node to prune if it has not contacted
				if metric.NumPings == 0 || (onlyScheduleActive && !active[*nodeState.GetID()]) {
					impl.State.SetStaleNode(nodeState.GetID())
				} else {
					nodeState.SetLastActive()
				}

				dbNode, ok := dbMap[*nodeState.GetID()]
				if !ok {
					jww.ERROR.Printf("Node [%s] exists in "+
						"state map but not in the database", nodeState.GetID())
				} else {
					if nodeState.GetLastActive().After(dbNode.LastActive) {
						toUpdate = append(toUpdate, nodeState.GetID())
					}

					if time.Since(dbNode.LastActive) > impl.params.pruneRetentionLimit {
						toPrune = append(toPrune, nodeState.GetID())
					}
				}

				// Store the NodeMetric
				if !onlyScheduleActive || active[*nodeState.GetID()] {
					err := storage.PermissioningDb.InsertNodeMetric(metric)
					if err != nil {
						jww.FATAL.Panicf(
							"Unable to store node metric: %+v", err)
					}
				}
			}

			// Update all the active nodes in the database
			err = storage.PermissioningDb.UpdateLastActive(toUpdate)
			if err != nil {
				jww.ERROR.Printf("TrackNodeMetrics: Could not update last active: %v", err)
			}

			if !RegParams.disableNDFPruning {
				// add disabled nodes to the prune list
				jww.DEBUG.Printf("Setting %d pruned nodes", len(toPrune))
				impl.State.SetPrunedNodes(toPrune)
				err := impl.State.UpdateNdf(impl.State.GetUnprunedNdf())
				if err != nil {
					jww.ERROR.Printf("Failed to regenerate the " +
						"NDF after changing pruning")
				}
			}
		}
	}
}

// GetActiveNodeIDs gets the active nodes from the database and returns the list
// of unmarshalled node IDs.
func GetActiveNodeIDs() (map[id.ID]bool, error) {
	nodes, err := storage.PermissioningDb.GetActiveNodes()
	if err != nil {
		return nil, errors.Errorf(getActiveNodesDbErr, err)
	}

	nodeIDs := make(map[id.ID]bool, len(nodes))
	for i, n := range nodes {

		nid, err := id.Unmarshal(n.Id)
		if err != nil {
			return nil, errors.Errorf(unmarshalActiveNodeDbErr, i, err)
		}

		nodeIDs[*nid] = true
	}

	return nodeIDs, nil
}
