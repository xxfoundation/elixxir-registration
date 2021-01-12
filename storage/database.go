////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles Database backend functionality
//+build database darwin linux windows

package storage

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	jww "github.com/spf13/jwalterweatherman"
	"time"
)

// Struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *gorm.DB // Stored Database connection
}

// Initialize the Database interface with Database backend
// Returns a Storage interface, Close function, and error
func NewDatabase(username, password, database, address,
	port string) (Storage, func() error, error) {

	var err error
	var db *gorm.DB
	//connect to the Database if the correct information is provided
	if address != "" && port != "" {
		// Create the Database connection
		connectString := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			address, port, username, database)
		// Handle empty Database password
		if len(password) > 0 {
			connectString += fmt.Sprintf(" password=%s", password)
		}
		db, err = gorm.Open("postgres", connectString)
	}

	// Return the map-backend interface
	// in the event there is a Database error or information is not provided
	if (address == "" || port == "") || err != nil {

		if err != nil {
			jww.WARN.Printf("Unable to initialize Database backend: %+v", err)
		} else {
			jww.WARN.Printf("Database backend connection information not provided")
		}

		defer jww.INFO.Println("Map backend initialized successfully!")

		return Storage{
			&MapImpl{
				applications: make(map[uint64]*Application),
				nodes:        make(map[string]*Node),
				nodeMetrics:  make(map[uint64]*NodeMetric),
				roundMetrics: make(map[uint64]*RoundMetric),
				clients:      make(map[string]*RegistrationCode),
				users:        make(map[string]bool),
				states:       make(map[string]string),
			}}, func() error { return nil }, nil
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
	models := []interface{}{
		&RegistrationCode{}, &User{}, &State{},
		&Application{}, &Node{}, &RoundMetric{}, &Topology{}, &NodeMetric{},
		&RoundError{},
	}
	for _, model := range models {
		err = db.AutoMigrate(model).Error
		if err != nil {
			return Storage{}, func() error { return nil }, err
		}
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return Storage{&DatabaseImpl{db: db}}, db.Close, nil

}
