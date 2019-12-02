////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles high level database control

package database

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	jww "github.com/spf13/jwalterweatherman"
	"sync"
)

// Struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *pg.DB // Stored database connection
}

//Fixme: This the real issue, need to remove this??

// Struct implementing the Database Interface with an underlying Map
type MapImpl struct {
	client map[string]*RegistrationCode
	node   map[string]*NodeInformation
	mut    sync.Mutex
}

type Permissioning struct {
	nodeRegistration
	clientRegistration
}

// Global variable for database interaction
var PermissioningDb Storage

type nodeRegistration interface {
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

type clientRegistration interface {
	// Inserts Client registration code with given number of uses
	InsertClientRegCode(code string, uses int) error
	// If Client registration code is valid, decrements remaining uses
	UseCode(code string) error
}

// Interface database storage operations
type Storage struct {
	clientRegistration
	nodeRegistration
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
func NewDatabase(username, password, database, address string) Storage {
	//TODO: Start up regCode database and make a map backend

	// Create the database connection
	db := pg.Connect(&pg.Options{
		User:         username,
		Password:     password,
		Database:     database,
		Addr:         address,
		MaxRetries:   10,
		MinIdleConns: 1,
	})

	_ = Storage{
		clientRegistration: clientRegistration(&MapImpl{
			client: make(map[string]*RegistrationCode),
		}),
		nodeRegistration: nodeRegistration(&MapImpl{
			node: make(map[string]*NodeInformation),
		})}
	// Initialize the schema
	jww.INFO.Println("Using database backend for Permissioning!")
	err := createSchema(db)
	if err != nil {
		// Return the map-backend interface
		// in the event there is a database error
		jww.ERROR.Printf("Unable to initialize database backend: %+v", err)
		jww.INFO.Println("Using map backend for UserRegistry!")
		return Storage{
			clientRegistration: clientRegistration(&MapImpl{
				client: make(map[string]*RegistrationCode),
			}),
			nodeRegistration: nodeRegistration(&MapImpl{
				node: make(map[string]*NodeInformation),
			})}
	}

	regCodeDb := &DatabaseImpl{
		db: db,
	}
	nodeMap := &MapImpl{
		client: make(map[string]*RegistrationCode),
	}

	jww.INFO.Println("Database/map backend initialized successfully!")
	return Storage{
		regCodeDb,
		nodeMap,
	}

}

// Create the database schema
func createSchema(db *pg.DB) error {
	for _, model := range []interface{}{&RegistrationCode{}} {
		err := db.CreateTable(model, &orm.CreateTableOptions{
			// Ignore create table if already exists?
			IfNotExists: true,
			// Create temporary table?
			Temp: false,
			// FKConstraints causes CreateTable to create foreign key constraints
			// for has one relations. ON DELETE hook can be added using tag
			// `sql:"on_delete:RESTRICT"` on foreign key field.
			FKConstraints: true,
			// Replaces PostgreSQL data type `text` with `varchar(n)`
			// Varchar: 255
		})
		if err != nil {
			// Return error if one comes up
			return err
		}
	}
	// No error, return nil
	return nil
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
