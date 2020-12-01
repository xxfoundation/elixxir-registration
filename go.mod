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
	gitlab.com/elixxir/comms v0.0.4-0.20201130214906-f87936e31f0f
	gitlab.com/elixxir/crypto v0.0.5-0.20201125005724-bcc603df02d3
	gitlab.com/elixxir/primitives v0.0.3-0.20201116174806-97f190989704
	gitlab.com/xx_network/comms v0.0.4-0.20201130190834-365ddae56e7b
	gitlab.com/xx_network/crypto v0.0.5-0.20201130191038-b4f77b62e2c2
	gitlab.com/xx_network/primitives v0.0.3-0.20201116234927-44e42fc91e7c
)

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1
