module gitlab.com/elixxir/registration

go 1.13

replace google.golang.org/grpc => github.com/grpc/grpc-go v1.27.1

require (
	github.com/fsnotify/fsnotify v1.4.9
	github.com/golang-collections/collections v0.0.0-20130729185459-604e922904d3
	github.com/jinzhu/gorm v1.9.16
	github.com/mitchellh/go-homedir v1.1.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	github.com/spf13/jwalterweatherman v1.1.0
	github.com/spf13/viper v1.7.1
	gitlab.com/elixxir/comms v0.0.0-20200911222931-62d34ebe9dff
	gitlab.com/elixxir/crypto v0.0.0-20200731174640-0503cf80524a
	gitlab.com/elixxir/primitives v0.0.0-20200708185800-a06e961280e6
	gitlab.com/xx_network/comms v0.0.0-20200911225302-05fdb4b165a3
)
