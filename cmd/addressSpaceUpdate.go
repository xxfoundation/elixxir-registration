////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/xx_network/primitives/ndf"
	"sort"
	"time"
)

var latestAddressSpace ndf.AddressSpace

// TrackAddressSpaceSizeUpdates starts a service that every interval checks
// storage for address space size updates and updates the NDF accordingly. The
// service runs until the quit channel is invoked.
func (m *RegistrationImpl) TrackAddressSpaceSizeUpdates(interval time.Duration,
	store storage.Storage, quit chan struct{}) {

	ticker := time.NewTicker(interval)
	var err error

	for {
		select {
		case <-quit:
			jww.INFO.Print("Stopping address space size update tracker.")
			return
		case <-ticker.C:
			latestAddressSpace, err = m.updateAddressSpace(latestAddressSpace,
				store)
			if err != nil {
				jww.FATAL.Panic(err)
			}
		}
	}
}

// updateAddressSpace checks if the address space in storage is newer than the
// one in memory and updates the address space list in the NDF and the state
// address space size if it is.
func (m *RegistrationImpl) updateAddressSpace(latest ndf.AddressSpace,
	store storage.Storage) (ndf.AddressSpace, error) {

	// Get latest ephemeral ID length from storage
	ephLen, err := store.GetLatestEphemeralLength()
	if err != nil {
		return ndf.AddressSpace{}, errors.Errorf(
			"failed to get latest address space size from storage: %+v", err)
	}

	// If there are updates, then get updated address space list,
	var addressSpaces []ndf.AddressSpace
	if latest.Timestamp.Before(ephLen.Timestamp) {
		addressSpaces, latest, err = GetAddressSpaceSizesFromStorage(store)
		if err != nil {
			return ndf.AddressSpace{}, errors.Errorf("failed to get latest "+
				"address space size list from storage: %+v", err)
		}

		m.State.SetAddressSpaceSize(uint32(latest.Size))

		jww.INFO.Printf("Address space size update found in database; set "+
			"state address space size to %d and updating the NDF with "+
			"changes (length of list %d).", latest.Size, len(addressSpaces))

		// Update the NDF
		m.NDFLock.Lock()
		updateNDF := m.State.GetUnprunedNdf()
		updateNDF.AddressSpace = addressSpaces
		m.State.UpdateInternalNdf(updateNDF)
		m.NDFLock.Unlock()
	}

	return latest, nil
}

// GetAddressSpaceSizesFromStorage returns a list of sorted address spaces and
// the newest addresses space from storage. An error is returned if no ephemeral
// ID lengths are found in storage.
func GetAddressSpaceSizesFromStorage(store storage.Storage) ([]ndf.AddressSpace,
	ndf.AddressSpace, error) {

	// Get list of ephemeral ID lengths from storage
	ephLens, err := store.GetEphemeralLengths()
	if err != nil {
		return nil, ndf.AddressSpace{}, err
	}

	// Sort ephemeral ID length list by timestamp
	sort.Slice(ephLens, func(i, j int) bool {
		return ephLens[i].Timestamp.Before(ephLens[j].Timestamp)
	})

	// Copy sorted ephemeral ID length list into address space list
	addressSpaces := make([]ndf.AddressSpace, len(ephLens))
	for i, ephLen := range ephLens {
		addressSpaces[i] = ndf.AddressSpace{
			Size:      ephLen.Length,
			Timestamp: ephLen.Timestamp,
		}
	}

	// Get the newest address space
	latestAddressSpace = addressSpaces[len(addressSpaces)-1]

	return addressSpaces, latestAddressSpace, nil
}
