////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"encoding/json"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"time"
)

// Error messages.
const (
	getActiveNodesDbErr      = "failed to get active node list from database: %+v"
	unmarshalActiveNodeDbErr = "failed to unmarshal active node ID #%d: %+v"
)

func TrackNodeMetrics(impl *RegistrationImpl, quitChan chan struct{}, nodeMetricInterval time.Duration) {
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
			var err error

			// Update whitelisted IDs
			whitelistedIds := make([]string, 0)
			if impl.params.WhitelistedIdsPath != "" {
				// Read file
				whitelistedIdsFile, err := utils.ReadFile(impl.params.WhitelistedIdsPath)
				if err != nil {
					jww.ERROR.Printf("Cannot read whitelisted IDs file (%s): %v",
						impl.params.WhitelistedIdsPath, err)
				}

				// Unmarshal JSON
				err = json.Unmarshal(whitelistedIdsFile, &whitelistedIds)
				if err != nil {
					jww.ERROR.Printf("Could not unmarshal whitelisted IDs: %v", err)
				}

			}

			// Update whitelisted IP addresses
			whitelistedIpAddresses := make([]string, 0)
			if impl.params.WhitelistedIpAddressPath != "" {
				// Read file
				whitelistedIpAddressesFile, err := utils.ReadFile(impl.params.WhitelistedIpAddressPath)
				if err != nil {
					jww.ERROR.Printf("Cannot read whitelisted IP addresses file (%s): %v",
						impl.params.WhitelistedIpAddressPath, err)
				}

				// Unmarshal JSON
				err = json.Unmarshal(whitelistedIpAddressesFile, &whitelistedIpAddresses)
				if err != nil {
					jww.ERROR.Printf("Could not unmarshal whitelisted IP addresses: %v", err)
				}

			}

			// Keep track of stale/pruned nodes
			// Set to true if pruned, false if stale
			toPrune := make(map[id.ID]bool)
			// List of nodes to update activity in Storage
			var toUpdate []*id.ID

			// Obtain active nodes
			var active map[id.ID]bool
			if onlyScheduleActive {
				active, err = GetActiveNodeIDs()
				if err != nil {
					jww.ERROR.Print(err)
				}
				jww.DEBUG.Printf("Found %d active nodes!", len(active))
			}

			// Iterate over the Node States
			nodeStates := impl.State.GetNodeMap().GetNodeStates()
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
					toPrune[*nodeState.GetID()] = false
				} else {
					nodeState.SetLastActive()
					toUpdate = append(toUpdate, nodeState.GetID())
				}
				if time.Since(nodeState.GetLastActive()) > impl.params.pruneRetentionLimit {
					toPrune[*nodeState.GetID()] = true
				}

				// Store the NodeMetric
				if !onlyScheduleActive || active[*nodeState.GetID()] {
					err = storage.PermissioningDb.InsertNodeMetric(metric)
					if err != nil {
						jww.FATAL.Panicf("Unable to store node metric: %+v", err)
					}
				}
			}

			// Update all the active nodes in the database
			err = storage.PermissioningDb.UpdateLastActive(toUpdate)
			if err != nil {
				jww.ERROR.Printf("TrackNodeMetrics: Could not update last active: %v", err)
			}

			if !impl.params.disableNDFPruning {
				// add disabled nodes to the prune list
				jww.DEBUG.Printf("Setting %d pruned nodes", len(toPrune))
				impl.NDFLock.Lock()
				impl.State.SetPrunedNodes(toPrune)
				currentNdf := impl.State.GetUnprunedNdf()
				currentNdf.WhitelistedIds = whitelistedIds
				currentNdf.WhitelistedIpAddresses = whitelistedIpAddresses
				err = impl.State.UpdateNdf(currentNdf)
				if err != nil {
					jww.ERROR.Printf("Failed to regenerate the " +
						"NDF after changing pruning")
				}
				impl.NDFLock.Unlock()
			}

			paramsCopy := impl.schedulingParams.SafeCopy()

			clientCutoff := impl.params.messageRetentionLimit + paramsCopy.RealtimeTimeout
			gatewayCutoff := impl.params.messageRetentionLimit + paramsCopy.RealtimeTimeout +
				2*(paramsCopy.PrecomputationTimeout+paramsCopy.RealtimeDelay+paramsCopy.RealtimeTimeout)

			earliestClientRound, _, clientErr := storage.PermissioningDb.
				GetEarliestRound(clientCutoff)

			earliestGwRound, earliestGwRoundTs, gatewayErr := storage.PermissioningDb.
				GetEarliestRound(gatewayCutoff)

			if clientErr != nil || gatewayErr != nil {
				if clientErr != nil && !errors.Is(clientErr, gorm.ErrRecordNotFound) {
					jww.ERROR.Printf("GetEarliestRound returned no records for client cutoff")
				} else if clientErr != nil {
					jww.ERROR.Printf("GetEarliestRound returned an error "+
						"for client cutoff: %v", clientErr)
				}

				if gatewayErr != nil && !errors.Is(gatewayErr, gorm.ErrRecordNotFound) {
					jww.ERROR.Printf("GetEarliestRound returned no records for gateway cutoff")
				} else if gatewayErr != nil {
					jww.ERROR.Printf("GetEarliestRound returned an error "+
						"for gateway cutoff: %v", gatewayErr)
				}
			} else {
				// If no errors, update impl
				impl.UpdateEarliestRound(earliestClientRound, earliestGwRound, earliestGwRoundTs)
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
