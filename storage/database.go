////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles high level database control

package storage

import (
	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage/node"
	"sync"
	"time"
)

// Struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *pg.DB // Stored database connection
}

// Struct implementing the Database Interface with an underlying Map
type MapImpl struct {
	client map[string]*RegistrationCode
	node   map[string]*Node
	user   map[string]bool
	mut    sync.Mutex
}

// Global variable for database interaction
var PermissioningDb Storage

type nodeRegistration interface {
	// If Node registration code is valid, add Node information
	InsertNode(id []byte, code, serverAddr, serverCert,
		gatewayAddress, gatewayCert string) error
	// Insert Node registration code into the database
	InsertNodeRegCode(regCode, order string) error
	// Get Node information for the given Node registration code
	GetNode(code string) (*Node, error)
}

type clientRegistration interface {
	// Inserts Client registration code with given number of uses
	InsertClientRegCode(code string, uses int) error
	// If Client registration code is valid, decrements remaining uses
	UseCode(code string) error
	// Gets User from the database
	GetUser(publicKey string) (*User, error)
	// Inserts User into the database
	InsertUser(publicKey string) error
}

// Interface database storage operations
type Storage struct {
	clientRegistration
	nodeRegistration
}

// Struct representing a RegistrationCode table in the database
type RegistrationCode struct {
	// Overwrite table name
	tableName struct{} `sql:"registration_codes,alias:rc"`

	// Registration code acts as the primary key
	Code string `sql:"pk"`
	// Remaining uses for the RegistrationCode
	RemainingUses int
}

// Struct representing the Node table in the database
type Node struct {
	// Overwrite table name
	tableName struct{} `sql:"nodes,alias:n"`

	// Registration code acts as the primary key
	Code string `sql:"pk"`
	// Node order string, this is a tag used by the algorithem
	Order string

	// Node ID
	Id []byte
	// Server IP address
	ServerAddress string
	// Gateway IP address
	GatewayAddress string
	// Node TLS public certificate in PEM string format
	NodeCertificate string
	// Gateway TLS public certificate in PEM string format
	GatewayCertificate string

	// Date/time that the node was registered
	DateRegistered time.Time
	// Node's network status
	Status uint8

	// ID of the Node's Application
	ApplicationId uint64 `pg:"notnull,unique"`

	// Node has one Application
	Application *Application
	// Node has many NodeMetrics
	NodeMetrics []*NodeMetric
	// Each Node participates in many Rounds
	RoundMetrics []*RoundMetric `pg:"many2many:node_to_round_metrics"`
}

// Struct representing the Node's Application table in the database
type Application struct {
	// Overwrite table name
	tableName struct{} `sql:"applications,alias:app"`

	// The Application's unique ID
	Id    uint64 `sql:"pk"`
	Name  string
	Url   string
	Blurb string

	// Location string for the Node
	Location string
	// Geographic bin of the Node's location
	GeoBin string
	// GPS location of the Node
	GpsLocation string

	// Social media
	Email     string
	Twitter   string
	Discord   string
	Instagram string
	Medium    string
}

// Struct representing Node Metrics table in the database
type NodeMetric struct {
	// Overwrite table name
	tableName struct{} `sql:"node_metrics,alias:nm"`

	// Auto-incrementing primary key
	Id uint64 `sql:"pk"`
	// Node has many NodeMetrics
	NodeId []byte `pg:"notnull"`
	// Start time of monitoring period
	StartTime time.Time `pg:"notnull"`
	// End time of monitoring period
	EndTime time.Time `pg:"notnull"`
	// Number of pings responded to during monitoring period
	NumPings uint64 `pg:"notnull"`
}

// Junction table for the many-to-many relationship between Nodes & RoundMetrics
type NodeToRoundMetrics struct {
	// Overwrite table name
	tableName struct{} `sql:"node_to_round_metrics,alias:nr"`

	// Composite primary key
	NodeId        []byte `sql:"pk"`
	RoundMetricId uint64 `sql:"pk"`

	// Order in the topology of a Node for a given Round
	Order uint8
}

// Struct representing Round Metrics table in the database
type RoundMetric struct {
	// Overwrite table name
	tableName struct{} `sql:"round_metrics,alias:rm"`

	// Auto-incrementing primary key
	Id uint64 `sql:"pk"`
	// Nullable error string, if one occurred
	Error string

	PrecompStart  time.Time `pg:"notnull"`
	PrecompEnd    time.Time `pg:"notnull"`
	RealtimeStart time.Time `pg:"notnull"`
	RealtimeEnd   time.Time `pg:"notnull"`
	Batchsize     uint32    `pg:"notnull"`

	// Each RoundMetric has many Nodes participating in each Round
	Topology []*Node `pg:"many2many:node_to_round_metrics"`
}

// Struct representing the User table in the database
type User struct {
	// Overwrite table name
	tableName struct{} `sql:"users,alias:u"`

	// User TLS public certificate in PEM string format
	PublicKey string `sql:"pk"`
}

// Initialize the Database interface with database backend
func NewDatabase(username, password, database, address string) Storage {
	// Create the database connection
	db := pg.Connect(&pg.Options{
		User:         username,
		Password:     password,
		Database:     database,
		Addr:         address,
		MaxRetries:   10,
		MinIdleConns: 1,
	})

	// Initialize the schema
	err := createSchema(db)
	if err != nil {
		// Return the map-backend interface
		// in the event there is a database error
		jww.ERROR.Printf("Unable to initialize database backend: %+v", err)
		jww.INFO.Println("Map backend initialized successfully!")
		return Storage{
			clientRegistration: clientRegistration(&MapImpl{
				client: make(map[string]*RegistrationCode),
				user:   make(map[string]bool),
			}),
			nodeRegistration: nodeRegistration(&MapImpl{
				node: make(map[string]*Node),
			})}
	}

	regCodeDb := &DatabaseImpl{
		db: db,
	}
	nodeDb := nodeRegistration(&DatabaseImpl{
		db: db,
	})

	jww.INFO.Println("Database backend initialized successfully!")
	return Storage{
		clientRegistration: regCodeDb,
		nodeRegistration:   nodeDb,
	}

}

// Create the database schema
func createSchema(db *pg.DB) error {
	// Register many to many model so ORM can better recognize m2m relation.
	orm.RegisterTable(&NodeToRoundMetrics{})

	// Create the models
	// Must be updated in order to create new models in the database
	models := []interface{}{
		&RegistrationCode{}, &User{}, &Node{},
		&Application{}, &NodeMetric{}, &RoundMetric{},
		&NodeToRoundMetrics{},
	}
	for _, model := range models {
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
func PopulateNodeRegistrationCodes(infos []node.Info) {
	for _, info := range infos {
		err := PermissioningDb.InsertNodeRegCode(info.RegCode, info.Order)
		if err != nil {
			jww.ERROR.Printf("Unable to populate Node registration code: %+v",
				err)
		}
	}
}
