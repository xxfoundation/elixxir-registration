////////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                           //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file                                                               //
////////////////////////////////////////////////////////////////////////////////

package cmd

import (
	"github.com/oschwald/geoip2-golang"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/region"
	"gitlab.com/xx_network/primitives/utils"
	"math/rand"
	"strconv"
	"sync/atomic"
	"time"
)

// Error messages.
const (
	parseIpErr        = "failed to parse node's address %q as an IP address"
	ipdbErr           = "failed to get node's country: %+v"
	ipdbNotRunningErr = "GeoIP2 database not running, reader probably closed"
	countryLookupErr  = "failed to get node's country: %+v"
	noBinErr          = "no bin associated with country %q"
	setDbSequenceErr  = "failed to set bin of node %s to %s"
	invalidFlagsErr   = "no GeoIP2 database provided and randomGeoBinning is " +
		"not set"
)

// setNodeBin assigns a region to each node. If a GeoIP2 database reader is
// supplied, then the region is assigned based off the node IP address's
// country. If randomGeoBinning in Params is set, then a random region is
// chosen; this is used for testing. If neither the reader or the flag is set,
// then an error is returned. The nodeIpAddr is the IP of of the node when it
// connects to permissioning; it is not the IP or domain name reported by the
// node.
func (m *RegistrationImpl) setNodeBin(n *node.State, nodeIpAddr string) error {
	// Get country code for node
	countryCode, err := getAddressCountry(nodeIpAddr, m.geoIPDB,
		m.params.randomGeoBinning, &m.geoIPDBStatus)

	// Get the geographical bin that the country belongs to
	bin, exists := region.GetCountryBin(countryCode)
	if !exists {
		return errors.Errorf(noBinErr, countryCode)
	}
	jww.DEBUG.Printf("Node %s is in bin %s", n.GetID(), bin)

	// Update sequence for the node in the database
	err = storage.PermissioningDb.UpdateNodeSequence(n.GetID(), bin.String())
	if err != nil {
		return errors.Errorf(setDbSequenceErr, n.GetID(), bin)
	}

	// Set the state ordering
	n.SetOrdering(bin.String())

	return nil
}

// getAddressCountry returns an alpha-2 country code for the address. Panics if
// randomGeoBinning is not set or a geoip2.Reader is not provided.
func getAddressCountry(ipAddr string, geoIPDB *geoip2.Reader,
	randomGeoBinning bool, geoipStatus *geoipStatus) (string, error) {
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
	} else if randomGeoBinning {
		// Assign a geographical bin from a randomly selected country
		return getRandomCountry(
			rand.New(rand.NewSource(time.Now().UnixNano()))), nil
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

// getRandomCountry returns a random alpha-2 country code.
func getRandomCountry(rng *rand.Rand) string {
	randomIndex := rng.Intn(region.CountryLen())
	return region.GetCountryList()[randomIndex]
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
