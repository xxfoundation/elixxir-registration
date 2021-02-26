module gitlab.com/elixxir/registration

go 1.13

require (
	github.com/armon/consul-api v0.0.0-20180202201655-eb2c6b5be1b6 // indirect
	github.com/denisenkom/go-mssqldb v0.0.0-20200428022330-06a60b6afbbc // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/jinzhu/gorm v1.9.12
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/lib/pq v1.5.2 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.1.0 // indirect
	github.com/spf13/cobra v1.1.1
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	github.com/ugorji/go v1.1.4 // indirect
	github.com/xordataexchange/crypt v0.0.3-0.20170626215501-b2862e3d0a77 // indirect
	gitlab.com/elixxir/client v1.2.1-0.20210222224029-4300043d7ce8
	gitlab.com/elixxir/comms v0.0.4-0.20210226003144-c355c2c144be
	gitlab.com/elixxir/crypto v0.0.7-0.20210226164631-dd11d922075b
	gitlab.com/elixxir/primitives v0.0.3-0.20210226174258-0b3abdb33fc3
	gitlab.com/xx_network/comms v0.0.4-0.20210226173933-8a1df6d9c9c9
	gitlab.com/xx_network/crypto v0.0.5-0.20210226174051-ac1ac369cb91
	gitlab.com/xx_network/primitives v0.0.4-0.20210226002915-98505d29e226
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
