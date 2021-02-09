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
	gitlab.com/elixxir/comms v0.0.4-0.20210209203849-4550368d6bc9
	gitlab.com/elixxir/crypto v0.0.7-0.20210208211519-5f5b427b29d7
	gitlab.com/elixxir/primitives v0.0.3-0.20210208211427-8140bdd93b0b
	gitlab.com/xx_network/comms v0.0.4-0.20210208222850-a4426ef99e37
	gitlab.com/xx_network/crypto v0.0.5-0.20210208211326-ba22f885317c
	gitlab.com/xx_network/primitives v0.0.4-0.20210208204253-ed1a9ef0c75e
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
