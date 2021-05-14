module gitlab.com/elixxir/registration

go 1.13

require (
	github.com/audiolion/ipip v1.0.0
	github.com/denisenkom/go-mssqldb v0.0.0-20200428022330-06a60b6afbbc // indirect
	github.com/fsnotify/fsnotify v1.4.9
	github.com/go-sql-driver/mysql v1.5.0 // indirect
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/jinzhu/gorm v1.9.12
	github.com/jinzhu/now v1.1.1 // indirect
	github.com/lib/pq v1.5.2 // indirect
	github.com/mattn/go-sqlite3 v2.0.3+incompatible // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/assertions v1.1.0 // indirect
	github.com/spf13/cobra v1.1.3
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/comms v0.0.4-0.20210514193223-7efa4c966f23
	gitlab.com/elixxir/crypto v0.0.7-0.20210514185043-7398614e9e43
	gitlab.com/elixxir/primitives v0.0.3-0.20210514174939-fbba435364ed
	gitlab.com/xx_network/comms v0.0.4-0.20210507215532-38ed97bd9365
	gitlab.com/xx_network/crypto v0.0.5-0.20210504210244-9ddabbad25fd
	gitlab.com/xx_network/primitives v0.0.4-0.20210504205835-db68f11de78a
	golang.org/x/net v0.0.0-20210315170653-34ac3e1c2000 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20210315173758-2651cd453018 // indirect
	google.golang.org/grpc v1.36.0 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
