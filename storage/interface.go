////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Handles low level Database structures and interfaces

package storage

import (
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/id"
	"sync"
	"time"
)

// Interface declaration for Storage methods
type database interface {
	// Permissioning methods
	UpsertState(state *State) error
	GetStateValue(key string) (string, error)
	InsertNodeMetric(metric *NodeMetric) error
	InsertRoundMetric(metric *RoundMetric, topology [][]byte) error
	InsertRoundError(roundId id.Round, errStr string) error
	GetLatestEphemeralLength() (*EphemeralLength, error)
	GetEphemeralLengths() ([]*EphemeralLength, error)
	InsertEphemeralLength(length *EphemeralLength) error

	// Node methods
	InsertApplication(application *Application, unregisteredNode *Node) error
	RegisterNode(id *id.ID, salt []byte, code, serverAddr, serverCert,
		gatewayAddress, gatewayCert string) error
	UpdateSalt(id *id.ID, salt []byte) error
	UpdateNodeAddresses(id *id.ID, nodeAddr, gwAddr string) error
	UpdateNodeSequence(id *id.ID, sequence string) error
	GetNode(code string) (*Node, error)
	GetNodeById(id *id.ID) (*Node, error)
	GetNodesByStatus(status node.Status) ([]*Node, error)
	GetActiveNodes() ([]*ActiveNode, error)

	// Client methods
	InsertClientRegCode(code string, uses int) error
	UseCode(code string) error
	GetUser(publicKey string) (*User, error)
	InsertUser(user *User) error
}

// Struct implementing the Database Interface with an underlying Map
type MapImpl struct {
	clients           map[string]*RegistrationCode
	nodes             map[string]*Node
	users             map[string]*User
	applications      map[uint64]*Application
	nodeMetrics       map[uint64]*NodeMetric
	nodeMetricCounter uint64
	roundMetrics      map[uint64]*RoundMetric
	states            map[string]string
	ephemeralLengths  map[uint8]*EphemeralLength
	activeNodes       map[id.ID]*ActiveNode
	mut               sync.Mutex
}

// Key-Value store used for persisting Permissioning State information
type State struct {
	Key   string `gorm:"primary_key"`
	Value string `gorm:"NOT NULL"`
}

// Enumerates Keys in the State table
const (
	UpdateIdKey = "UpdateId"
	RoundIdKey  = "RoundId"
	EllipticKey = "EllipticKey"
)

// Struct representing a RegistrationCode table in the Database
type RegistrationCode struct {
	// Registration code acts as the primary key
	Code string `gorm:"primary_key"`
	// Remaining uses for the RegistrationCode
	RemainingUses int
}

// Struct representing the User table in the Database
type User struct {
	// User TLS public certificate in PEM string format
	PublicKey string `gorm:"primary_key"`
	// User reception key in PEM string format
	ReceptionKey string `gorm:"NOT NULL;UNIQUE"`
	// Timestamp in which user registered with permissioning
	RegistrationTimestamp time.Time `gorm:"NOT NULL"`
}

// Struct representing the Node's Application table in the Database
type Application struct {
	// The Application's unique ID
	Id uint64 `gorm:"primary_key;AUTO_INCREMENT:false"`
	// Each Application has one Node
	Node Node `gorm:"foreignkey:ApplicationId"`

	// Node information
	Name  string
	Url   string
	Blurb string
	Other string

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
	Forum     string
	Email     string
	Twitter   string
	Discord   string
	Instagram string
	Medium    string
}

// Struct representing the ActiveNode table in the Database
type ActiveNode struct {
	Id []byte `gorm:"primary_key"`
}

// Struct representing the Node table in the Database
type Node struct {
	// Registration code acts as the primary key
	Code string `gorm:"primary_key"`
	// Node order string, this is a tag used by the algorithm
	Sequence string

	// Unique Node ID
	Id []byte `gorm:"UNIQUE_INDEX;default: null"`
	// Salt used for generation of Node ID
	Salt []byte
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

// Struct representing Node Metrics table in the Database
type NodeMetric struct {
	// Auto-incrementing primary key (Do not set)
	Id uint64 `gorm:"primary_key;AUTO_INCREMENT:true"`
	// Node has many NodeMetrics
	NodeId []byte `gorm:"INDEX;NOT NULL;type:bytea REFERENCES nodes(Id)"`
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
	RoundMetricId uint64 `gorm:"INDEX;primary_key;type:bigint REFERENCES round_metrics(Id)"`

	// Order in the topology of a Node for a given Round
	Order uint8 `gorm:"NOT NULL"`
}

// Struct representing Round Metrics table in the Database
type RoundMetric struct {
	// Unique ID of the round as assigned by the network
	Id uint64 `gorm:"primary_key;AUTO_INCREMENT:false"`

	// Round timestamp information
	PrecompStart  time.Time `gorm:"NOT NULL"`
	PrecompEnd    time.Time `gorm:"NOT NULL;INDEX;"`
	RealtimeStart time.Time `gorm:"NOT NULL"`
	RealtimeEnd   time.Time `gorm:"NOT NULL;INDEX;"` // Index for TPS calc
	BatchSize     uint32    `gorm:"NOT NULL"`

	// Each RoundMetric has many Nodes participating in each Round
	Topologies []Topology `gorm:"foreignkey:RoundMetricId;association_foreignkey:Id"`

	// Each RoundMetric can have many Errors in each Round
	RoundErrors []RoundError `gorm:"foreignkey:RoundMetricId;association_foreignkey:Id"`
}

// Struct representing Round Errors table in the Database
type RoundError struct {
	// Auto-incrementing primary key (Do not set)
	Id uint64 `gorm:"primary_key;AUTO_INCREMENT:true"`

	// ID of the round for a given run of the network
	RoundMetricId uint64 `gorm:"INDEX;NOT NULL;type:bigint REFERENCES round_metrics(Id)"`

	// String of error that occurred during the Round
	Error string `gorm:"NOT NULL"`
}

// Struct representing the validity period of an ephemeral ID length
type EphemeralLength struct {
	Length    uint8     `gorm:"primary_key;AUTO_INCREMENT:false"`
	Timestamp time.Time `gorm:"NOT NULL;UNIQUE"`
}

// Initialize the database interface with Map backend
func NewMap() Storage {
	defer jww.INFO.Println("Map backend initialized successfully!")
	return Storage{
		&MapImpl{
			applications:     make(map[uint64]*Application),
			nodes:            make(map[string]*Node),
			nodeMetrics:      make(map[uint64]*NodeMetric),
			roundMetrics:     make(map[uint64]*RoundMetric),
			clients:          make(map[string]*RegistrationCode),
			users:            make(map[string]*User),
			states:           make(map[string]string),
			ephemeralLengths: make(map[uint8]*EphemeralLength),
			activeNodes:      make(map[id.ID]*ActiveNode),
		}}
}

// Adds Client registration codes to the Database
func PopulateClientRegistrationCodes(codes []string, uses int) {
	for _, code := range codes {
		err := PermissioningDb.InsertClientRegCode(code, uses)
		if err != nil {
			jww.ERROR.Printf("Unable to populate Client registration code: %+v",
				err)
		}
	}
}

// Adds Node registration codes to the Database
func PopulateNodeRegistrationCodes(infos []node.Info) {
	// TODO: This will eventually need to be updated to intake applications too
	i := 1
	for _, info := range infos {
		err := PermissioningDb.InsertApplication(&Application{
			Id: uint64(i),
		}, &Node{
			Code:          info.RegCode,
			Sequence:      info.Order,
			ApplicationId: uint64(i),
		})
		if err != nil {
			jww.ERROR.Printf("Unable to populate Node registration code: %+v",
				err)
		}
		i++
	}
}
