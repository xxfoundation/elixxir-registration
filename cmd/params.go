////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

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
	Address                    string
	CertPath                   string
	KeyPath                    string
	FullNdfOutputPath          string
	SignedPartialNdfOutputPath string
	NsCertPath                 string
	NsAddress                  string
	WhitelistedIdsPath         string
	WhitelistedIpAddressPath   string

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

	// How long rounds will be tracked by gateways.
	// Rounds (and messages as an extension)
	// prior to this period are not guaranteed to be delivered to clients.
	// Expects duration in"h". (Defaults to 1 weeks (168 hours)
	messageRetentionLimit    time.Duration
	messageRetentionLimitMux sync.Mutex

	// Specs on rate limiting clients
	leakedCapacity uint32
	leakedTokens   uint32
	leakedDuration uint64
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

func (p *Params) GetMessageRetention() time.Duration {
	p.messageRetentionLimitMux.Lock()
	defer p.messageRetentionLimitMux.Unlock()
	return p.messageRetentionLimit
}
