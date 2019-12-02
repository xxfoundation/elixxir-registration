////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles high level database control

package database

import (
	"github.com/go-pg/pg"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
)

// Struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *pg.DB // Stored database connection
}

// Struct implementing the Database Interface with an underlying Map
type MapImpl struct {
	client map[string]*RegistrationCode
	node   map[string]*NodeInformation
	mut    sync.Mutex
}

// Global variable for database interaction
var PermissioningDb Database

// Interface database storage operations
type Database interface {
	// Inserts Client registration code with given number of uses
	InsertClientRegCode(code string, uses int) error
	// If Client registration code is valid, decrements remaining uses
	UseCode(code string) error
	// If Node registration code is valid, add Node information
	InsertNode(id []byte, code, serverCert,
		gatewayAddress, gatewayCert string) error
	// Insert Node registration code into the database
	InsertNodeRegCode(code string) error
	// Count the number of Nodes currently registered
	CountRegisteredNodes() (int, error)
	// Get Node information for the given Node registration code
	GetNode(code string) (*NodeInformation, error)
}

// Struct representing a RegistrationCode table in the database
type RegistrationCode struct {
	// Overwrite table name
	tableName struct{} `sql:"registration_codes,alias:registration_codes"`

	// Registration code acts as the primary key
	Code string `sql:",pk"`
	// Remaining uses for the RegistrationCode
	RemainingUses int
}

// Struct representing the Node Information table in the database
type NodeInformation struct {
	// Overwrite table name
	tableName struct{} `sql:"nodes,alias:nodes"`

	// Registration code acts as the primary key
	Code string `sql:",pk"`
	// Node ID
	Id []byte
	// Gateway IP address
	GatewayAddress string
	// Node TLS public certificate in PEM string format
	NodeCertificate string
	// Gateway TLS public certificate in PEM string format
	GatewayCertificate string
}

// Initialize the Database interface with database backend
func NewDatabase(username, password, database, address string) Database {
	jww.INFO.Println("Using map backend for UserRegistry!")
	return Database(&MapImpl{
		client: make(map[string]*RegistrationCode),
		node:   make(map[string]*NodeInformation),
	})
}

// Adds Client registration codes to the database
func PopulateClientRegistrationCodes(codes []string, uses int) {
	for _, code := range codes {
		err := PermissioningDb.InsertClientRegCode(code, uses)
		if err != nil {
			jww.ERROR.Printf("Unable to populate Client registration code: %+v",
				err)
		}
	}
}

// Adds Node registration codes to the database
func PopulateNodeRegistrationCodes(codes []string) {
	for _, code := range codes {
		err := PermissioningDb.InsertNodeRegCode(code)
		if err != nil {
			jww.ERROR.Printf("Unable to populate Node registration code: %+v",
				err)
		}
	}
}
