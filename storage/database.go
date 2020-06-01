////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles high level database control

package storage

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/primitives/id"
	"gitlab.com/elixxir/registration/storage/node"
	"sync"
	"time"
)

// Struct implementing the Database Interface with an underlying DB
type DatabaseImpl struct {
	db *gorm.DB // Stored database connection
}

// Struct implementing the Database Interface with an underlying Map
type MapImpl struct {
	clients            map[string]*RegistrationCode
	nodes              map[string]*Node
	users              map[string]bool
	applications       map[uint64]*Application
	nodeMetrics        map[uint64]*NodeMetric
	nodeMetricCounter  uint64
	roundMetrics       map[uint64]*RoundMetric
	roundMetricCounter uint64
	mut                sync.Mutex
}

// Global variable for database interaction
var PermissioningDb Storage

type nodeRegistration interface {
	// If Node registration code is valid, add Node information
	RegisterNode(id *id.ID, code, serverAddr, serverCert,
		gatewayAddress, gatewayCert string) error
	// Get Node information for the given Node registration code
	GetNode(code string) (*Node, error)
	// Return all nodes in storage with the given Status
	GetNodesByStatus(status node.Status) ([]*Node, error)
	// Insert Application object along with associated unregistered Node
	InsertApplication(application Application, unregisteredNode Node) error
	// Insert NodeMetric object
	InsertNodeMetric(metric NodeMetric) error
	// Insert RoundMetric object
	InsertRoundMetric(metric RoundMetric, topology [][]byte) error
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
	// Registration code acts as the primary key
	Code string `gorm:"primary_key"`
	// Remaining uses for the RegistrationCode
	RemainingUses int
}

// Struct representing the User table in the database
type User struct {
	// User TLS public certificate in PEM string format
	PublicKey string `gorm:"primary_key"`
}

// Struct representing the Node's Application table in the database
type Application struct {
	// The Application's unique ID
	Id uint64 `gorm:"primary_key"`
	// Each Application has one Node
	Node Node `gorm:"foreignkey:ApplicationId"`

	// Node information
	Name  string
	Url   string
	Blurb string

	// Location string for the Node
	Location string
	// Geographic bin of the Node's location
	GeoBin string
	// GPS location of the Node
	GpsLocation string
	// Specifies the team the node was assigned
	Team string
	// Specifies which network the node is in
	Network string

	// Social media
	Email     string
	Twitter   string
	Discord   string
	Instagram string
	Medium    string
}

// Struct representing the Node table in the database
type Node struct {
	// Registration code acts as the primary key
	Code string `gorm:"primary_key"`
	// Node order string, this is a tag used by the algorithm
	Sequence string

	// Unique Node ID
	Id []byte `gorm:"UNIQUE_INDEX;default: null"`
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
	Status uint8 `gorm:"NOT NULL"`

	// Unique ID of the Node's Application
	ApplicationId uint64 `gorm:"UNIQUE_INDEX;NOT NULL;type:bigint REFERENCES applications(id)"`

	// Each Node has many Node Metrics
	NodeMetrics []NodeMetric `gorm:"foreignkey:NodeId;association_foreignkey:Id"`

	// Each Node participates in many Rounds
	Topologies []Topology `gorm:"foreignkey:NodeId;association_foreignkey:Id"`
}

// Struct representing Node Metrics table in the database
type NodeMetric struct {
	// Auto-incrementing primary key (Do not set)
	Id uint64 `gorm:"primary_key;AUTO_INCREMENT"`
	// Node has many NodeMetrics
	NodeId []byte `gorm:"NOT NULL;type:bytea REFERENCES nodes(Id)"`
	// Start time of monitoring period
	StartTime time.Time `gorm:"NOT NULL"`
	// End time of monitoring period
	EndTime time.Time `gorm:"NOT NULL"`
	// Number of pings responded to during monitoring period
	NumPings uint64 `gorm:"NOT NULL"`
}

// Junction table for the many-to-many relationship between Nodes & RoundMetrics
type Topology struct {
	// Composite primary key
	NodeId        []byte `gorm:"primary_key;type:bytea REFERENCES nodes(Id)"`
	RoundMetricId uint64 `gorm:"primary_key;type:bigint REFERENCES round_metrics(Id)"`

	// Order in the topology of a Node for a given Round
	Order uint8 `gorm:"NOT NULL"`
}

// Struct representing Round Metrics table in the database
type RoundMetric struct {
	// Auto-incrementing primary key (Do not set)
	Id uint64 `gorm:"primary_key;AUTO_INCREMENT"`
	// Nullable error string, if one occurred
	Error string

	PrecompStart  time.Time `gorm:"NOT NULL"`
	PrecompEnd    time.Time `gorm:"NOT NULL"`
	RealtimeStart time.Time `gorm:"NOT NULL"`
	RealtimeEnd   time.Time `gorm:"NOT NULL"`
	BatchSize     uint32    `gorm:"NOT NULL"`

	// Each RoundMetric has many Nodes participating in each Round
	Topologies []Topology `gorm:"foreignkey:RoundMetricId;association_foreignkey:Id"`
}

// Initialize the Database interface with database backend
func NewDatabase(username, password, database, address, port string) (Storage,
	error) {

	var err error
	var db *gorm.DB
	//connect to the database if the correct information is provided
	if address != "" && port != "" {
		// Create the database connection
		connectString := fmt.Sprintf(
			"host=%s port=%s user=%s dbname=%s sslmode=disable",
			address, port, username, database)
		// Handle empty database password
		if len(password) > 0 {
			connectString += fmt.Sprintf(" password=%s", password)
		}
		db, err = gorm.Open("postgres", connectString)
	}

	// Return the map-backend interface
	// in the event there is a database error or information is not provided
	if (address == "" || port == "") || err != nil {

		if err != nil {
			jww.WARN.Printf("Unable to initialize database backend: %+v", err)
		} else {
			jww.WARN.Printf("Database backend connection information not provided")
		}

		defer jww.INFO.Println("Map backend initialized successfully!")

		return Storage{
			clientRegistration: clientRegistration(&MapImpl{
				clients: make(map[string]*RegistrationCode),
				users:   make(map[string]bool),
			}),
			nodeRegistration: nodeRegistration(&MapImpl{
				applications: make(map[uint64]*Application),
				nodes:        make(map[string]*Node),
				nodeMetrics:  make(map[uint64]*NodeMetric),
				roundMetrics: make(map[uint64]*RoundMetric),
			})}, nil
	}

	// Initialize the database logger
	db.SetLogger(jww.DEBUG)
	db.LogMode(true)

	// Initialize the database schema
	// WARNING: Order is important. Do not change without database testing
	models := []interface{}{
		&RegistrationCode{}, &User{},
		&Application{}, &Node{}, &RoundMetric{}, &Topology{}, &NodeMetric{},
	}
	for _, model := range models {
		err = db.AutoMigrate(model).Error
		if err != nil {
			return Storage{}, err
		}
	}

	// Build the interface
	di := &DatabaseImpl{
		db: db,
	}

	jww.INFO.Println("Database backend initialized successfully!")
	return Storage{
		clientRegistration: di,
		nodeRegistration:   di,
	}, nil

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
	// TODO: This will eventually need to be updated to intake applications too
	for i, info := range infos {
		err := PermissioningDb.InsertApplication(Application{
			Id: uint64(i),
		}, Node{
			Code:          info.RegCode,
			Sequence:      info.Order,
			ApplicationId: uint64(i),
		})
		if err != nil {
			jww.ERROR.Printf("Unable to populate Node registration code: %+v",
				err)
		}
	}
}
