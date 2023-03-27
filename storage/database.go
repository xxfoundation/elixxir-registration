////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Handles Database backend functionality
//go:build !stateless
// +build !stateless

package storage

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/pkg/errors"
	jww "github.com/spf13/jwalterweatherman"
	"time"
)

const postgresConnectString = "host=%s port=%s user=%s dbname=%s sslmode=disable"
const sqliteDatabasePath = "file:%s?mode=memory&cache=shared"

// Struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *gorm.DB // Stored Database connection
}

// Initialize the database interface with Database backend
// Returns a Storage interface, Close function, and error
func NewDatabase(username, password, database, address,
	port string) (Storage, func() error, error) {

	var err error
	var db *gorm.DB
	//connect to the Database if the correct information is provided
	var useSqlite bool
	var connString, dialect string
	// Connect to the database if the correct information is provided
	if address != "" && port != "" {
		// Create the database connection
		connString = fmt.Sprintf(
			postgresConnectString,
			address, port, username, database)
		// Handle empty database password
		if len(password) > 0 {
			connString += fmt.Sprintf(" password=%s", password)
		}
		dialect = "postgres"
	} else {
		useSqlite = true
		jww.WARN.Printf("Database backend connection information not provided")
		connString = fmt.Sprintf(sqliteDatabasePath, database)
		dialect = "sqlite3"
	}

	// Create the database connection
	db, err = gorm.Open(dialect, connString)
	if err != nil {
		return Storage{}, nil, errors.Errorf("Unable to initialize database backend: %+v", err)
	}

	if useSqlite {
		// Enable foreign keys because they are disabled in SQLite by default
		if err = db.Exec("PRAGMA foreign_keys = ON", nil).Error; err != nil {
			return Storage{}, nil, errors.WithMessage(err, "Failed to enable foreign keys")
		}

		// Enable Write Ahead Logging to enable multiple DB connections
		if err = db.Exec("PRAGMA journal_mode = WAL;", nil).Error; err != nil {
			return Storage{}, nil, errors.WithMessage(err, "Failed to enable journal mode")
		}
	}

	// Initialize the Database logger
	db.SetLogger(jww.TRACE)
	db.LogMode(true)

	// SetMaxIdleConns sets the maximum number of connections in the idle connection pool.
	db.DB().SetMaxIdleConns(10)
	// SetMaxOpenConns sets the maximum number of open connections to the Database.
	db.DB().SetMaxOpenConns(100)
	// SetConnMaxLifetime sets the maximum amount of time a connection may be reused.
	db.DB().SetConnMaxLifetime(24 * time.Hour)

	// Initialize the Database schema
	// WARNING: Order is important. Do not change without Database testing
	var models []interface{}
	if useSqlite {
		models = []interface{}{
			&State{}, &Application{}, &Node{}, &RoundMetricAlt{}, &Topology{}, &NodeMetric{},
			&RoundError{}, EphemeralLength{}, ActiveNode{}, GeoBin{},
		}
	} else {
		models = []interface{}{
			&State{}, &Application{}, &Node{}, &RoundMetric{}, &Topology{}, &NodeMetric{},
			&RoundError{}, EphemeralLength{}, ActiveNode{}, GeoBin{},
		}
	}

	for _, model := range models {
		err = db.AutoMigrate(model).Error
		if err != nil {
			return Storage{}, func() error { return nil }, errors.WithMessage(err, "Failed to AutoMigrate schema")
		}
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return Storage{&DatabaseImpl{db: db}}, db.Close, nil

}
