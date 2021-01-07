module gitlab.com/elixxir/registration

go 1.13

require (
	github.com/denisenkom/go-mssqldb v0.0.0-20200428022330-06a60b6afbbc // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
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
	gitlab.com/elixxir/comms v0.0.4-0.20210107184523-c6ed72fd848c
	gitlab.com/elixxir/crypto v0.0.7-0.20210107184400-5c3e52a35758
	gitlab.com/elixxir/primitives v0.0.3-0.20210107183456-9cf6fe2de1e5
	gitlab.com/xx_network/comms v0.0.4-0.20210107184305-7bb439c08ded
	gitlab.com/xx_network/crypto v0.0.5-0.20210107183440-804e0f8b7d22
	gitlab.com/xx_network/primitives v0.0.4-0.20210106014326-691ebfca3b07
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
