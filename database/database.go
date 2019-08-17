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

// Struct implementing the Database Interface with an underlying Map
type MapImpl struct {
	client map[string]*RegistrationCode
	node   map[string]*NodeInformation
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
	InsertNode(id []byte, code, address, nodeCert, gatewayCert string) error
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

	// Ensure an empty NodeInformation table
	err := db.DropTable(&NodeInformation{},
		&orm.DropTableOptions{IfExists: true})
	if err != nil {
		// If an error is thrown with the database, run with a map backend
		jww.INFO.Println("Using map backend for UserRegistry!")
		return Database(&MapImpl{
			client: make(map[string]*RegistrationCode),
			node:   make(map[string]*NodeInformation),
		})
	}

	// Initialize the schema
	jww.INFO.Println("Using database backend for Permissioning!")
	err = createSchema(db)
	if err != nil {
		jww.FATAL.Panicf("Unable to initialize database backend: %+v", err)
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
