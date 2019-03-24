////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles the database ORM for registration codes

package database

import (
	"errors"
	"fmt"
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	jww "github.com/spf13/jwalterweatherman"
	"time"
)

// Global variable for database interaction
var RegCodes RegistrationStorage

// Interface for Registration Code storage operations
type RegistrationStorage interface {
	// Inserts  code with given number of uses
	InsertCode(code string, uses int) error
	// If code is valid, decrements remaining uses
	UseCode(code string) error
}

// Struct implementing the RegistrationStorage Interface with an underlying DB
type RegistrationDatabase struct {
	db *pg.DB // Stored database connection
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

// Initialize the RegistrationStorage interface with database backend
func NewRegistrationStorage(username, password,
	database, address string) RegistrationStorage {

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
		jww.FATAL.Panicf("Unable to use database backend for Registration"+
			" Codes: %s", err)
	}

	// Return the database-backed RegistrationStorage interface
	jww.INFO.Println("Using database backend for Registration Codes!")
	return RegistrationStorage(&RegistrationDatabase{
		db: db,
	})
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

// Adds some dummy registration codes to the database
func PopulateDummyRegistrationCodes() {
	err := RegCodes.InsertCode("AAAA", 100)
	if err != nil {
		jww.ERROR.Printf("Unable to populate dummy codes: %s", err)
	}
}

// Inserts code with given number of uses
func (m *RegistrationDatabase) InsertCode(code string, uses int) error {
	err := m.db.Insert(&RegistrationCode{
		Code:          code,
		RemainingUses: uses,
	})
	return err
}

// If code is valid, decrements remaining uses
func (m *RegistrationDatabase) UseCode(code string) error {

	// Look up given registration code
	regCode := RegistrationCode{Code: code}
	err := m.db.Select(&regCode)
	if err != nil {
		// Unable to find code, return error
		return err
	}

	if regCode.RemainingUses < 1 {
		// Code has no remaining uses, return error
		return errors.New(fmt.Sprintf("Code %s has no remaining uses", code))
	}

	// Decrement remaining uses by one
	regCode.RemainingUses -= 1
	err = m.db.Update(&regCode)

	// Return error, if any
	return err
}
