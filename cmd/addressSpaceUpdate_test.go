package cmd

import (
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/xx_network/primitives/ndf"
	"reflect"
	"testing"
	"time"
)

// Happy path.
func TestRegistrationImpl_updateAddressSpace(t *testing.T) {
	// Create list of address spaces to add to storage
	addressSpaces, latest := makeSortedAddressSpaces(16)

	// Initialise storage
	store, _, err := storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf("Failed to create new database: %+v", err)
	}

	// Set the global so that storage.NewState works
	storage.PermissioningDb = store

	// Add the address spaces to storage
	for i, as := range addressSpaces {
		err = store.InsertEphemeralLength(&storage.EphemeralLength{Length: as.Size, Timestamp: as.Timestamp})
		if err != nil {
			t.Errorf("Failed to insert address space as ephemeral length (%d): %+v", i, err)
		}
	}

	// Create a new state
	state, err := storage.NewState(getTestKey(), 8, "")
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}

	// Set state address space size
	state.SetAddressSpaceSize(uint32(addressSpaces[0].Size))

	// Add only the first address space to the NDF and update
	updateNDF := state.GetUnprunedNdf()
	updateNDF.AddressSpace = addressSpaces[:1]
	err = state.UpdateNdf(updateNDF)
	if err != nil {
		t.Errorf("Unable to update NDF: %+v", err)
	}
	m := RegistrationImpl{State: state}

	// Update NDF
	testLatest, err := m.updateAddressSpace(addressSpaces[0], store)
	if err != nil {
		t.Errorf("updateAddressSpace() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(m.State.GetUnprunedNdf().AddressSpace, addressSpaces) {
		t.Errorf("updateAddressSpace() did not correctly update the NDF."+
			"\nexpected: %+v\nreceived: %+v", addressSpaces, m.State.GetUnprunedNdf().AddressSpace)
	}

	if !reflect.DeepEqual(latest, testLatest) {
		t.Errorf("updateAddressSpace() did not return the expected latest address space."+
			"\nexpected: %+v\nreceived: %+v", latest, testLatest)
	}

	if m.State.GetAddressSpaceSize() != uint32(latest.Size) {
		t.Errorf("updateAddressSpace() did not set the correct state addres space size."+
			"\nexpected: %d\nreceived: %d", latest.Size, m.State.GetAddressSpaceSize())
	}
}

// Happy path: no updates are available and the NDF is not changed.
func TestRegistrationImpl_updateAddressSpace_NoUpdates(t *testing.T) {
	// Create list of address spaces to add to storage
	addressSpaces, latest := makeSortedAddressSpaces(16)

	// Initialise storage
	store, _, err := storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf("Failed to create new database: %+v", err)
	}

	// Set the global so that storage.NewState works
	storage.PermissioningDb = store

	// Add the address spaces to storage
	for i, as := range addressSpaces {
		err = store.InsertEphemeralLength(&storage.EphemeralLength{Length: as.Size, Timestamp: as.Timestamp})
		if err != nil {
			t.Errorf("Failed to insert address space as ephemeral length (%d): %+v", i, err)
		}
	}

	// Create a new state
	state, err := storage.NewState(getTestKey(), 8, "")
	if err != nil {
		t.Errorf("Unable to create state: %+v", err)
	}

	// Add only the latest address spaces to the NDF and update
	updateNDF := state.GetUnprunedNdf()
	updateNDF.AddressSpace = []ndf.AddressSpace{latest}
	err = state.UpdateNdf(updateNDF)
	if err != nil {
		t.Errorf("Unable to update NDF: %+v", err)
	}
	m := RegistrationImpl{State: state}

	// Update NDF
	testLatest, err := m.updateAddressSpace(latest, store)
	if err != nil {
		t.Errorf("updateAddressSpace() returned an error: %+v", err)
	}

	// Check that the NDF has not been updated
	if !reflect.DeepEqual(m.State.GetUnprunedNdf().AddressSpace, []ndf.AddressSpace{latest}) {
		t.Errorf("updateAddressSpace() did not correctly update the NDF."+
			"\nexpected: %+v\nreceived: %+v", []ndf.AddressSpace{latest}, m.State.GetUnprunedNdf().AddressSpace)
	}

	if !reflect.DeepEqual(latest, testLatest) {
		t.Errorf("updateAddressSpace() did not return the expected latest address space."+
			"\nexpected: %+v\nreceived: %+v", latest, testLatest)
	}
}

// Error path: storage has no address spaces.
func TestRegistrationImpl_updateAddressSpace_NoStorage(t *testing.T) {
	// Initialise storage
	store, _, err := storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf("Failed to create new database: %+v", err)
	}

	// Add only the first address space to the NDF and update
	m := RegistrationImpl{}

	// Update NDF
	_, err = m.updateAddressSpace(ndf.AddressSpace{}, store)
	if err == nil {
		t.Error("updateAddressSpace() did not return an error when there were " +
			"no ephemeral ID lengths in storage.")
	}
}

// Happy path.
func Test_GetAddressSpaceSizesFromStorage(t *testing.T) {
	// Create list of address spaces to add to storage
	expected, expectedLatest := makeSortedAddressSpaces(16)

	// Initialise storage
	store, _, err := storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf("Failed to create new database: %+v", err)
	}

	// Add the address spaces to storage
	for i, as := range expected {
		err = store.InsertEphemeralLength(&storage.EphemeralLength{Length: as.Size, Timestamp: as.Timestamp})
		if err != nil {
			t.Errorf("Failed to insert address space as ephemeral length (%d): %+v", i, err)
		}
	}

	addressSpaces, latest, err := GetAddressSpaceSizesFromStorage(store)
	if err != nil {
		t.Errorf("GetAddressSpaceSizesFromStorage() returned an error: %+v", err)
	}

	if !reflect.DeepEqual(expected, addressSpaces) {
		t.Errorf("GetAddressSpaceSizesFromStorage() did not return the expected address spaces."+
			"\nexpected: %+v\nreceived: %+v", expected, addressSpaces)
	}

	if !reflect.DeepEqual(expectedLatest, latest) {
		t.Errorf("GetAddressSpaceSizesFromStorage() did not return the expected latest address space."+
			"\nexpected: %+v\nreceived: %+v", expectedLatest, latest)
	}
}

// Error path: storage has no address spaces.
func Test_GetAddressSpaceSizesFromStorage_EmptyStorageError(t *testing.T) {
	// Initialise storage
	store, _, err := storage.NewDatabase("", "", "", "", "")
	if err != nil {
		t.Errorf("Failed to create new database: %+v", err)
	}

	_, _, err = GetAddressSpaceSizesFromStorage(store)
	if err == nil {
		t.Errorf("GetAddressSpaceSizesFromStorage() did not return an error " +
			"when storage should have been empty.")
	}
}

// makeSortedAddressSpaces creates a list of ndf.AddressSpace sorted by their
// timestamp with the newest last.
func makeSortedAddressSpaces(n uint8) ([]ndf.AddressSpace, ndf.AddressSpace) {
	addressSpaces := make([]ndf.AddressSpace, n)
	initSize := uint8(0)
	initTime := time.Now()
	var addressSpace ndf.AddressSpace

	for i := range addressSpaces {
		addressSpace = ndf.AddressSpace{
			Size:      initSize,
			Timestamp: initTime.UTC(),
		}
		addressSpaces[i] = addressSpace
		initSize++
		initTime = initTime.Add(8760 * time.Hour)
	}

	return addressSpaces, addressSpace
}
