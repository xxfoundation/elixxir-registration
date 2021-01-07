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
	"time"
)

// Params object for reading in configuration data
type Params struct {
	Address               string
	CertPath              string
	KeyPath               string
	NdfOutputPath         string
	NsCertPath            string
	NsAddress             string
	cmix                  ndf.Group
	e2e                   ndf.Group
	publicAddress         string
	schedulingKillTimeout time.Duration
	closeTimeout          time.Duration
	minimumNodes          uint32
	udbId                 []byte
	udbCertPath           string
	udbAddress            string
	minGatewayVersion     version.Version
	minServerVersion      version.Version
	disableGatewayPing    bool
	// User registration can take userRegCapacity registrations in userRegLeakPeriod period of time
	userRegCapacity   uint32
	userRegLeakPeriod time.Duration
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
