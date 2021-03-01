module gitlab.com/elixxir/registration

go 1.13

require (
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
	gitlab.com/elixxir/client v1.2.1-0.20210222224029-4300043d7ce8
	gitlab.com/elixxir/comms v0.0.4-0.20210301173501-38cf2a1fc999
	gitlab.com/elixxir/crypto v0.0.7-0.20210226194937-5d641d5a31bc
	gitlab.com/elixxir/primitives v0.0.3-0.20210301171428-ad09b913b569
	gitlab.com/xx_network/comms v0.0.4-0.20210226194929-ea05928f74b7
	gitlab.com/xx_network/crypto v0.0.5-0.20210226194923-5f470e2a2533
	gitlab.com/xx_network/primitives v0.0.4-0.20210301172945-82f5d4248c04
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
