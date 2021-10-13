///////////////////////////////////////////////////////////////////////////////
// Copyright © 2020 xx network SEZC                                          //
//                                                                           //
// Use of this source code is governed by a license that can be found in the //
// LICENSE file                                                              //
///////////////////////////////////////////////////////////////////////////////

// Contains Params-related functionality

package cmd

import (
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/xx_network/primitives/ndf"
	"sync"
	"time"
)

// Params object for reading in configuration data
type Params struct {
	Address                  string
	CertPath                 string
	KeyPath                  string
	NdfOutputPath            string
	NsCertPath               string
	NsAddress                string
	WhitelistedIdsPath       string
	WhitelistedIpAddressPath string

	cmix                  ndf.Group
	e2e                   ndf.Group
	publicAddress         string
	schedulingKillTimeout time.Duration
	closeTimeout          time.Duration
	minimumNodes          uint32
	udbId                 []byte
	udbDhPubKey           []byte
	udbCertPath           string
	udbAddress            string
	minGatewayVersion     version.Version
	minServerVersion      version.Version
	minClientVersion      version.Version
	addressSpaceSize      uint8
	allowLocalIPs         bool
	disableGeoBinning     bool
	blockchainGeoBinning  bool
	disablePing           bool
	onlyScheduleActive    bool
	enableBlockchain      bool

	disableNDFPruning bool

	geoIPDBFile string

	clientRegistrationAddress string

	versionLock sync.RWMutex

	// How long offline nodes remain in the NDF. If a node is
	// offline past this duration the node is cleared from the
	// NDF. Expects duration in"h". (Defaults to 1 week (168 hours)
	pruneRetentionLimit time.Duration

	//
}

// toGroup takes a group represented by a map of string to string,
// then uses the prime and generator to create an ndf group object.
func toGroup(grp map[string]string) (*ndf.Group, error) {
	jww.DEBUG.Printf("Group is: %v", grp)
	pStr, pOk := grp["prime"]
	gStr, gOk := grp["generator"]

	if !gOk || !pOk {
		return nil, errors.Errorf("Invalid Group Config "+
			"(prime: %v, generator: %v", pOk, gOk)
	}
	return &ndf.Group{Prime: pStr, Generator: gStr}, nil
}
