module gitlab.com/elixxir/registration

go 1.13

require (
	github.com/denisenkom/go-mssqldb v0.0.0-20200428022330-06a60b6afbbc // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/golang/protobuf v1.4.2 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20200217142428-fce0ec30dd00 // indirect
	github.com/jinzhu/gorm v1.9.12
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/lib/pq v1.5.2 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.3.0 // indirect
	github.com/pelletier/go-toml v1.7.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.1.0 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.6.1 // indirect
	github.com/zeebo/blake3 v0.0.4 // indirect
	gitlab.com/elixxir/comms v0.0.0-20200820194546-0881d5e2fd51
	gitlab.com/elixxir/crypto v0.0.0-20200731174640-0503cf80524a
	gitlab.com/elixxir/primitives v0.0.0-20200731184040-494269b53b4d
	gitlab.com/xx_network/comms v0.0.0-20200731174953-53980d88816e
	gitlab.com/xx_network/primitives v0.0.0-20200805165129-a0cd942ed608
	gitlab.com/xx_network/ring v0.0.2 // indirect
	golang.org/x/crypto v0.0.0-20200707235045-ab33eee955e0 // indirect
	golang.org/x/net v0.0.0-20200707034311-ab3426394381 // indirect
	golang.org/x/sys v0.0.0-20200625212154-ddb9806d33ae // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200709005830-7a2ca40e9dc3 // indirect
	google.golang.org/grpc v1.30.0 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)

replace (
	gitlab.com/xx_network/collections/ring => gitlab.com/xx_network/collections/ring.git v0.0.1
	google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
)
