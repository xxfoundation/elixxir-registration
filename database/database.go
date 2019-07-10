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
	"time"
)

// Struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *pg.DB // Stored database connection
}

// Global variable for database interaction
var PermissioningDb Database

// Interface database storage operations
type Database interface {
	// Inserts client registration code with given number of uses
	InsertClientRegCode(code string, uses int) error
	// If client registration code is valid, decrements remaining uses
	UseCode(code string) error
	// If node registration code is valid, add node information
	InsertNode(id []byte, code, address, nodeCert, gatewayCert string) error
	// Insert node registration code into the database
	InsertNodeRegCode(code string) error
	// Obtain the full internal node topology
	GetRegisteredNodes() ([]NodeInformation, error)
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
	// IP address
	Address string
	// Node TLS public certificate in PEM string format
	NodeCertificate string
	// Gateway TLS public certificate in PEM string format
	GatewayCertificate string
}

// Initialize the Database interface with database backend
func NewDatabase(username, password, database, address string) Database {

	// Create the database connection
	db := pg.Connect(&pg.Options{
		User:        username,
		Password:    password,
		Database:    database,
		Addr:        address,
		PoolSize:    1,
		MaxRetries:  10,
		PoolTimeout: time.Duration(2) * time.Minute,
		IdleTimeout: time.Duration(10) * time.Minute,
		MaxConnAge:  time.Duration(1) * time.Hour,
	})

	// Attempt to connect to the database and initialize the schema
	err := createSchema(db)
	if err != nil {
		jww.FATAL.Panicf("Unable to initialize database backend: %s", err)
	}

	// Return the database-backed Database interface
	jww.INFO.Println("Database backend initialized successfully!")
	return Database(&DatabaseImpl{
		db: db,
	})
}

// Create the database schema
func createSchema(db *pg.DB) error {
	for _, model := range []interface{}{&RegistrationCode{}, &NodeInformation{}} {
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

// Adds dummy registration codes to the database
func PopulateDummyRegistrationCodes() {
	// Client registration codes
	err := PermissioningDb.InsertClientRegCode("AAAA", 100)
	if err != nil {
		jww.ERROR.Printf("Unable to populate dummy client registration codes"+
			": %s", err)
	}

	// Node registration codes
	codes := []string{"ZZZZ", "YYYY", "XXXX"}
	for _, code := range codes {
		err := PermissioningDb.InsertNodeRegCode(code)
		if err != nil {
			jww.ERROR.Printf("Unable to populate node registration codes: %s",
				err)
		}
	}
}
