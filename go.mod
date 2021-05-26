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
	gitlab.com/elixxir/comms v0.0.4-0.20210526170645-e7a77c41f345
	gitlab.com/elixxir/crypto v0.0.7-0.20210526002540-1fb51df5b4b2
	gitlab.com/elixxir/primitives v0.0.3-0.20210526002350-b9c947fec050
	gitlab.com/xx_network/comms v0.0.4-0.20210526002311-2b5a66af0eac
	gitlab.com/xx_network/crypto v0.0.5-0.20210526002149-9c08ccb202be
	gitlab.com/xx_network/primitives v0.0.4-0.20210525232109-3f99a04adcfd
	gitlab.com/xx_network/ring v0.0.3-0.20210525230516-6a1185d19af8 // indirect
	golang.org/x/net v0.0.0-20210315170653-34ac3e1c2000 // indirect
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20210315173758-2651cd453018 // indirect
	google.golang.org/grpc v1.36.0 // indirect
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
