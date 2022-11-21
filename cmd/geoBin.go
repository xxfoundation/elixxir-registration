////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"compress/gzip"
	"fmt"
	"github.com/jinzhu/gorm"
	"gitlab.com/xx_network/primitives/region"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/utils"
)

// Error messages.
const (
	parseIpErr        = "failed to parse node's address %q as an IP address"
	ipdbErr           = "failed to get node's country: %+v"
	ipdbNotRunningErr = "GeoIP2 database not running, reader probably closed"
	countryLookupErr  = "failed to get node's country: %+v"
	setDbSequenceErr  = "failed to set bin of node %s to %s"
	invalidFlagsErr   = "no GeoIP2 database provided and randomGeoBinning is " +
		"not set"

	geoIPUpdateInterval       = 7 * 24 * time.Hour
	geoIPLastModifiedStateKey = "geoIPDBLastModified"
)

// startUpdateGeoIPDB initializes a thread which updates the held geoIP database
// on a regular interval (currently 7 days)
func (m *RegistrationImpl) startUpdateGeoIPDB() (chan bool, error) {
	// Load last modified from storage if it exists
	lastModified, err := storage.PermissioningDb.GetStateValue(geoIPLastModifiedStateKey)
	if err != nil {
		if !strings.Contains(err.Error(), gorm.ErrRecordNotFound.Error()) &&
			!strings.Contains(err.Error(), "Unable to locate state for key") {
			jww.WARN.Printf("Could not find state value for geoIP last modified timestamp: %+v", err)
		}
	} else {
		err = m.geoIPLastModified.UnmarshalText([]byte(lastModified))
		if err != nil {
			jww.ERROR.Printf("Failed to unmarshal last modified from storage")
		}
	}

	// Perform initial update pass for geoIPDB
	updated, err := m.updateGeoIPDB()
	if err != nil {
		return nil, err
	}

	if updated {
		err = m.updateAllNodeGeos()
		if err != nil {
			return nil, errors.WithMessage(err, "Failed to update all node geo bins after initial geoIPDB update")
		}
	}

	// Start update thread
	stop := make(chan bool)
	go func() {
		interval := time.NewTimer(geoIPUpdateInterval)
		for {
			select {
			case <-interval.C:
				updated, err := m.updateGeoIPDB()
				if err != nil {
					jww.ERROR.Printf("Regular update of GeoIP database failed: %+v", err)
				}

				if updated {
					err = m.updateAllNodeGeos()
					if err != nil {
						jww.ERROR.Printf("Failed to update all node geo bins after initial geoIPDB update: %+v", err)
					}
				}
			case <-stop:
				jww.INFO.Println("GeoIP update thread received stop signal")
				m.geoIPDBLock.Lock()
				m.geoIPDBStatus.ToStopped()
				err = m.geoIPDB.Close()
				m.geoIPDBLock.Unlock()
				if err != nil {
					jww.ERROR.Printf("Failed to close GeoIPDB when stop signal received: %+v", err)
				}
				return
			}
		}
	}()
	return stop, err
}

// updateGeoIPDB handles the logic for updating the geoIPDB held by registration
// It first checks the latest headers, and if the server's file is more recent
// than ours, downloads and replaces the current geoIP database
func (m *RegistrationImpl) updateGeoIPDB() (bool, error) {
	if m.params.geoIPDBFile == "" || m.params.geoIPDBUrl == "" {
		return false, errors.Errorf("Cannot update geoIPDB without both geoIPDBFile and geoIPDBUrl set")
	}
	// Get latest timestamp for geoIPDB on server
	timestamp, _, err := getLatestHeaders(m.params.geoIPDBUrl)
	if err != nil {
		return false, err
	}
	serverLastModified, err := time.Parse(time.RFC1123, timestamp)
	// If last modified on server is not later than ours, return now
	if !serverLastModified.After(m.geoIPLastModified) {
		fmt.Println("hi")
		return false, nil
	}

	// Get latest GeoIPDB from permalink
	newGeoIPDB, err := getGeoIPDB(m.params.geoIPDBUrl)
	if err != nil {
		return false, err
	}

	// Write to file, use a suffix for now rather than immdeiately replacing
	newFilePath := m.params.geoIPDBFile + ".new"
	err = os.WriteFile(newFilePath, newGeoIPDB, os.ModePerm)
	if err != nil {
		return false, err
	}

	// Attempt to open the newly downloaded database
	newDB, err := geoip2.Open(newFilePath)
	if err != nil {
		return false, errors.WithMessage(err, "Failed to update geoIPDB")
	}
	jww.TRACE.Printf("Successfully downloaded new GeoIP database version %d.%d", newDB.Metadata().BinaryFormatMajorVersion, newDB.Metadata().BinaryFormatMinorVersion)
	err = newDB.Close()
	if err != nil {
		return false, err
	}

	// Now we take the lock and swap the databases
	m.geoIPDBLock.Lock()
	defer m.geoIPDBLock.Unlock()
	// Close out the currently open db
	if m.geoIPDB != nil {
		m.geoIPDBStatus.ToStopped()
		err = m.geoIPDB.Close()
		if err != nil {
			return false, errors.WithMessage(err, "Failed to close geoIPDB")
		}
		m.geoIPDB = nil
	}

	// Swap old file for new, keeping the old one around until we're sure the replace worked
	oldFilePath := m.params.geoIPDBFile + ".old"
	err = os.Rename(m.params.geoIPDBFile, oldFilePath)
	if err != nil {
		return false, err
	}
	err = os.Rename(newFilePath, m.params.geoIPDBFile)
	if err != nil {
		// If we fail to move the new DB into place, try to restore the old one
		err2 := os.Rename(oldFilePath, m.params.geoIPDBFile)
		if err2 != nil {
			jww.ERROR.Printf("Failed to restore old geoIPDB file after error: %+v", err)
		}
		m.geoIPDB, err2 = geoip2.Open(m.params.geoIPDBFile)
		if err2 != nil {
			jww.FATAL.Panicf("Failed to restore previous geoIPDB after error: %+v", err)
		}
		return false, err
	}

	m.geoIPDB, err = geoip2.Open(m.params.geoIPDBFile)
	if err != nil {
		// If we fail to open the new database, try to restore the old one
		err2 := os.Rename(oldFilePath, m.params.geoIPDBFile)
		if err2 != nil {
			jww.ERROR.Printf("Failed to replace old geoIPDB file after error: %+v", err)
		}
		m.geoIPDB, err2 = geoip2.Open(m.params.geoIPDBFile)
		if err2 != nil {
			jww.FATAL.Panicf("Failed to restore previous geoIPDB after error: %+v", err)
		}
		return false, err
	}

	// Finally, remove the old db file
	err = os.Remove(oldFilePath)
	if err != nil {
		return false, err
	}

	m.geoIPDBStatus.ToRunning()
	lastModifiedBytes, err := serverLastModified.MarshalText()
	if err != nil {
		return true, err
	}
	err = storage.PermissioningDb.UpsertState(&storage.State{
		Key:   geoIPLastModifiedStateKey,
		Value: string(lastModifiedBytes),
	})
	if err != nil {
		return true, errors.WithMessage(err, "Failed to update geoIP last modified state")
	}
	m.geoIPLastModified = serverLastModified

	return true, nil
}

// getLatestHeaders gets the latest lastModified and contentDisposition headers from the passed in maxmind permalink
// These should be used to ensure we only download a new database when it has updates
func getLatestHeaders(url string) (string, string, error) {
	resp, err := http.Head(url)
	if err != nil {
		return "", "", errors.WithMessage(err, "Failed to get GeoIPDB headers")
	}
	lastModified := resp.Header.Get("last-modified")
	contentDisposition := resp.Header.Get("content-disposition")
	return lastModified, contentDisposition, nil
}

// getGeoIPDB retrieves the latest compressed maxmind geoIP database from the passed in permalink
func getGeoIPDB(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to get GeoIPDB from permalink")
	}
	decompresser, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to create gzip reader for response body")
	}
	decompressed, err := io.ReadAll(decompresser)
	if err != nil {
		return nil, errors.WithMessage(err, "Failed to decompress GeoIPDB file")
	}
	return decompressed, nil
}

// updateAllNodeGeos calls setNodeSequence on all nodes in the state node map
func (m *RegistrationImpl) updateAllNodeGeos() error {
	var errs []error
	curStates := m.State.GetNodeMap().GetNodeStates()
	for _, n := range curStates {
		err := m.setNodeSequence(n, n.GetNodeAddresses())
		if err != nil {
			err = errors.Errorf("Error updating ip of node %s: %+v", n.GetID().String(), err)
			jww.ERROR.Println(err)
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Errorf("Failed to register with %d nodes", len(errs))
	}
	return nil
}

func (m *RegistrationImpl) setNodeGeos(n *node.State, location, geo_bin, gps_location string) error {

	return storage.PermissioningDb.UpdateGeoIP(n.GetAppID(), location, geo_bin, gps_location)
}

// setNodeSequence assigns a country code to each node
func (m *RegistrationImpl) setNodeSequence(n *node.State, nodeIpAddr string) error {
	var countryCode, countryName, city, gps string
	var geobin region.GeoBin
	var err error
	var ok bool
	// Get country code for node
	if m.params.disableGeoBinning {
		countryCode = n.GetOrdering()
	} else {
		m.geoIPDBLock.RLock()
		countryCode, err = getAddressCountry(nodeIpAddr, m.geoIPDB, &m.geoIPDBStatus)
		if err != nil {
			m.geoIPDBLock.RUnlock()
			return errors.WithMessage(err, "Failed to get country for address")
		}
		city, err = getAddressCity(nodeIpAddr, m.geoIPDB, &m.geoIPDBStatus)
		if err != nil {
			m.geoIPDBLock.RUnlock()
			return errors.WithMessage(err, "Failed to get city for address")
		}
		gps, err = getAddressCoords(nodeIpAddr, m.geoIPDB, &m.geoIPDBStatus)
		if err != nil {
			m.geoIPDBLock.RUnlock()
			return errors.WithMessage(err, "Failed to get gps for address")
		}
		geobin, ok = region.GetCountryBin(countryCode)
		if !ok {
			m.geoIPDBLock.RUnlock()
			return errors.WithMessage(err, "Could not get bin for country code")
		}
		countryName, err = lookupCountryName(nodeIpAddr, m.geoIPDB)
		if err != nil {
			m.geoIPDBLock.RUnlock()
			return errors.WithMessage(err, "Could not get country name")
		}
		m.geoIPDBLock.RUnlock()
	}

	// Update sequence for the node in the database
	err = storage.PermissioningDb.UpdateNodeSequence(n.GetID(), countryCode)
	if err != nil {
		return errors.Errorf(setDbSequenceErr, n.GetID(), countryCode)
	}

	// Generate the location string (exclude city if none is found)
	location := countryName
	if city != "" {
		location = city + ", " + location
	}

	err = storage.PermissioningDb.UpdateGeoIP(
		n.GetAppID(), location, geobin.String(), gps)

	// Set the state ordering
	n.SetOrdering(countryCode)
	return nil
}

// getAddressCountry returns an alpha-2 country code for the address. Panics if
// randomGeoBinning is not set or a geoip2.Reader is not provided.
func getAddressCountry(ipAddr string, geoIPDB *geoip2.Reader, geoipStatus *geoipStatus) (string, error) {
	if geoIPDB != nil {
		// Return an error if the status is not set to running (meaning the
		// reader has been closed)
		if !geoipStatus.IsRunning() {
			return "", errors.New(ipdbNotRunningErr)
		}

		// Get country code for the country of the node's IP address
		countryCode, err := lookupCountry(ipAddr, geoIPDB)
		if err != nil {
			return "", errors.Errorf(countryLookupErr, err)
		}
		return countryCode, nil
	}

	err := errors.New(invalidFlagsErr)
	jww.FATAL.Panic("Cannot get node bins: " + err.Error())

	return "", err
}

// lookupCountry returns the alpha-2 country code of where the address is
// located as found in the GeoIP2 database.
func lookupCountry(ipAddr string, geoIPDB *geoip2.Reader) (string, error) {
	// Parse the IP string into a net.IP object
	ip := utils.ParseIP(ipAddr)
	if ip == nil {
		return "", errors.Errorf(parseIpErr, ipAddr)
	}

	// Get the node's country from its IP address via the GeoIP2 database
	country, err := geoIPDB.Country(ip)
	if err != nil {
		return "", errors.Errorf(ipdbErr, err)
	}

	// Return the two letter alpha-2 country code
	return country.Country.IsoCode, nil
}

// getAddressCity returns the city for an IP address. Panics if
// randomGeoBinning is not set or a geoip2.Reader is not provided.
func getAddressCity(ipAddr string, geoIPDB *geoip2.Reader, geoipStatus *geoipStatus) (string, error) {
	if geoIPDB != nil {
		// Return an error if the status is not set to running (meaning the
		// reader has been closed)
		if !geoipStatus.IsRunning() {
			return "", errors.New(ipdbNotRunningErr)
		}

		// Get country code for the country of the node's IP address
		city, err := lookupCity(ipAddr, geoIPDB)
		if err != nil {
			return "", errors.Errorf(countryLookupErr, err)
		}
		return city, nil
	}

	err := errors.New(invalidFlagsErr)
	jww.FATAL.Panic("Cannot get node bins: " + err.Error())

	return "", err
}

// lookupCity returns the city of where the address is
// located as found in the GeoIP2 database.
func lookupCity(ipAddr string, geoIPDB *geoip2.Reader) (string, error) {
	// Parse the IP string into a net.IP object
	ip := utils.ParseIP(ipAddr)
	if ip == nil {
		return "", errors.Errorf(parseIpErr, ipAddr)
	}

	// Get the node's country from its IP address via the GeoIP2 database
	country, err := geoIPDB.City(ip)
	if err != nil {
		return "", errors.Errorf(ipdbErr, err)
	}

	// Return the city
	return country.City.Names["en"], nil
}

func lookupCountryName(ipAddr string, geoIPDB *geoip2.Reader) (string, error) {
	// Parse the IP string into a net.IP object
	ip := utils.ParseIP(ipAddr)
	if ip == nil {
		return "", errors.Errorf(parseIpErr, ipAddr)
	}

	// Get the node's country from its IP address via the GeoIP2 database
	country, err := geoIPDB.Country(ip)
	if err != nil {
		return "", errors.Errorf(ipdbErr, err)
	}

	// Return the city
	return country.Country.Names["en"], nil
}

// getAddressCoords returns the coords for an IP address. Panics if
// randomGeoBinning is not set or a geoip2.Reader is not provided.
func getAddressCoords(ipAddr string, geoIPDB *geoip2.Reader, geoipStatus *geoipStatus) (string, error) {
	if geoIPDB != nil {
		// Return an error if the status is not set to running (meaning the
		// reader has been closed)
		if !geoipStatus.IsRunning() {
			return "0.0, 0.0", errors.New(ipdbNotRunningErr)
		}

		// Get country code for the country of the node's IP address
		latitude, longitude, err := lookupCoords(ipAddr, geoIPDB)
		if err != nil {
			return "0.0, 0.0", errors.Errorf(countryLookupErr, err)
		}
		return fmt.Sprintf("%f, %f", latitude, longitude), nil
	}

	err := errors.New(invalidFlagsErr)
	jww.FATAL.Panic("Cannot get node bins: " + err.Error())

	return "0.0, 0.0", err
}

// lookupCoords returns the coords of where the address approx. is
// located as found in the GeoIP2 database. Latitude, then Longitude
func lookupCoords(ipAddr string, geoIPDB *geoip2.Reader) (float64, float64, error) {
	// Parse the IP string into a net.IP object
	ip := utils.ParseIP(ipAddr)
	if ip == nil {
		return 0.0, 0.0, errors.Errorf(parseIpErr, ipAddr)
	}

	// Get the node's country from its IP address via the GeoIP2 database
	record, err := geoIPDB.City(ip)
	if err != nil {
		return 0.0, 0.0, errors.Errorf(ipdbErr, err)
	}

	// Return the city
	return record.Location.Latitude, record.Location.Longitude, nil
}

// geoipStatus signals the status of the GeoIP2 database reader. It should be
// used as an atomic.
type geoipStatus uint32

// Possible values for the geoipStatus.
const (
	geoipNotStarted geoipStatus = iota
	geoipRunning
	geoipStopped
)

// ToRunning changes the status to running.
func (s *geoipStatus) ToRunning() {
	atomic.StoreUint32((*uint32)(s), uint32(geoipRunning))
}

// ToStopped changes the status to stopped.
func (s *geoipStatus) ToStopped() {
	atomic.StoreUint32((*uint32)(s), uint32(geoipStopped))
}

// IsRunning returns true if the status is currently running.
func (s *geoipStatus) IsRunning() bool {
	return s.GetStatus() == geoipRunning
}

// GetStatus returns the current status
func (s *geoipStatus) GetStatus() geoipStatus {
	return geoipStatus(atomic.LoadUint32((*uint32)(s)))
}

// String returns the string representation of the status. This functions
// satisfies the fmt.Stringer interface.
func (s geoipStatus) String() string {
	switch s {
	case geoipNotStarted:
		return "not started"
	case geoipRunning:
		return "running"
	case geoipStopped:
		return "stopped"
	default:
		return "INVALID STATUS " + strconv.Itoa(int(s))
	}
}
