////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger

package cmd

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/scheduling"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

var (
	cfgFile              string
	logLevel             uint // 0 = info, 1 = debug, >1 = trace
	noTLS                bool
	RegParams            Params
	ClientRegCodes       []string
	clientVersion        string
	clientVersionLock    sync.RWMutex
	disablePermissioning bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "registration",
	Short: "Runs a registration server for cMix",
	Long:  `This server provides registration functions on cMix`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cmixMap := viper.GetStringMapString("groups.cmix")
		e2eMap := viper.GetStringMapString("groups.e2e")

		cmix, err := toGroup(cmixMap)
		if err != nil {
			jww.FATAL.Panicf("Failed to create cMix group: %+v", err)
		}
		e2e, err := toGroup(e2eMap)
		if err != nil {
			jww.FATAL.Panicf("Failed to create E2E group: %+v", err)
		}

		// Parse config file options
		certPath := viper.GetString("certPath")
		keyPath := viper.GetString("keyPath")
		localAddress := fmt.Sprintf("0.0.0.0:%d", viper.GetInt("port"))
		ndfOutputPath := viper.GetString("ndfOutputPath")
		setClientVersion(viper.GetString("clientVersion"))
		ipAddr := viper.GetString("publicAddress")
		//Get Notification Server address and cert Path
		nsCertPath := viper.GetString("nsCertPath")
		nsAddress := viper.GetString("nsAddress")
		publicAddress := fmt.Sprintf("%s:%d", ipAddr, viper.GetInt("port"))

		maxRegistrationAttempts := viper.GetUint64("maxRegistrationAttempts")
		if maxRegistrationAttempts == 0 {
			maxRegistrationAttempts = defaultMaxRegistrationAttempts
		}

		registrationCountDuration := viper.GetDuration("registrationCountDuration")
		if registrationCountDuration == 0 {
			registrationCountDuration = defaultRegistrationCountDuration
		}

		// Set up database connection
		storage.PermissioningDb, err = storage.NewDatabase(
			viper.GetString("dbUsername"),
			viper.GetString("dbPassword"),
			viper.GetString("dbName"),
			viper.GetString("dbAddress"),
		)
		if err != nil {
			jww.FATAL.Panicf("Unable to initialize storage: %+v", err)
		}

		// Populate Node registration codes into the database
		RegCodesFilePath := viper.GetString("regCodesFilePath")
		regCodeInfos, err := node.LoadInfo(RegCodesFilePath)
		if err != nil {
			jww.FATAL.Panicf("Failed to load registration codes from the "+
				"file %s: %+v", RegCodesFilePath, err)
		}
		storage.PopulateNodeRegistrationCodes(regCodeInfos)

		ClientRegCodes = viper.GetStringSlice("clientRegCodes")
		storage.PopulateClientRegistrationCodes(ClientRegCodes, 1000)

		udbId := make([]byte, 32)
		udbId[len(udbId)-1] = byte(viper.GetInt("udbID"))

		//load the scheduling params file as a string
		SchedulingConfigPath := viper.GetString("schedulingConfigPath")
		SchedulingConfig, err := utils.ReadFile(SchedulingConfigPath)
		if err != nil {
			jww.FATAL.Panicf("Could not load Scheduling Config file: %v", err)
		}

		// Populate params
		RegParams = Params{
			Address:                   localAddress,
			CertPath:                  certPath,
			KeyPath:                   keyPath,
			NdfOutputPath:             ndfOutputPath,
			cmix:                      *cmix,
			e2e:                       *e2e,
			publicAddress:             publicAddress,
			NsAddress:                 nsAddress,
			NsCertPath:                nsCertPath,
			maxRegistrationAttempts:   maxRegistrationAttempts,
			registrationCountDuration: registrationCountDuration,
			udbId:                     udbId,
			minimumNodes:              viper.GetUint32("minimumNodes"),
		}

		jww.INFO.Println("Starting Permissioning Server...")

		// Start registration server
		impl, err := StartRegistration(RegParams)
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}

		jww.INFO.Printf("Waiting for for %v nodes to register so "+
			"rounds can start", RegParams.minimumNodes)

		<-impl.beginScheduling
		jww.INFO.Printf("Minnimum number of nodes %v have registered,"+
			"begining scheduling and round creation", RegParams.minimumNodes)

		// Begin scheduling algorithm
		go func() {
			err = scheduling.Scheduler(SchedulingConfig, impl.State)
			jww.FATAL.Panicf("Scheduling Algorithm exited: %s", err)
		}()

		// Block forever to prevent the program ending
		select {}
	},
}

// Execute adds all child commands to the root command and sets flags
// appropriately.  This is called by main.main(). It only needs to
// happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

// init is the initialization function for Cobra which defines commands
// and flags.
func init() {
	// NOTE: The point of init() is to be declarative.
	// There is one init in each sub command. Do not put variable declarations
	// here, and ensure all the Flags are of the *P variety, unless there's a
	// very good reason not to have them as local params to sub command."
	cobra.OnInitialize(initConfig, initLog)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.
	rootCmd.Flags().UintVarP(&logLevel, "logLevel", "l", 1,
		"Level of debugging to display. 0 = info, 1 = debug, >1 = trace")

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c",
		"", "Sets a custom config file path")

	rootCmd.Flags().BoolVar(&noTLS, "noTLS", false,
		"Runs without TLS enabled")

	rootCmd.Flags().BoolVarP(&disablePermissioning, "disablePermissioning", "",
		false, "Disables registration server checking for ndf updates")

}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	//Use default config location if none is passed
	if cfgFile == "" {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			jww.ERROR.Println(err)
			os.Exit(1)
		}

		cfgFile = home + "/.elixxir/registration.yaml"

	}

	validConfig := true
	f, err := os.Open(cfgFile)
	if err != nil {
		jww.ERROR.Printf("Unable to open config file (%s): %+v", cfgFile, err)
		validConfig = false
	}
	_, err = f.Stat()
	if err != nil {
		jww.ERROR.Printf("Invalid config file (%s): %+v", cfgFile, err)
		validConfig = false
	}
	err = f.Close()
	if err != nil {
		jww.ERROR.Printf("Unable to close config file (%s): %+v", cfgFile, err)
		validConfig = false
	}

	// Set the config file if it is valid
	if validConfig {
		// Set the config path to the directory containing the config file
		// This may increase the reliability of the config watching, somewhat
		cfgDir, _ := path.Split(cfgFile)
		viper.AddConfigPath(cfgDir)

		viper.SetConfigFile(cfgFile)
		viper.AutomaticEnv() // read in environment variables that match

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err != nil {
			jww.ERROR.Printf("Unable to parse config file (%s): %+v", cfgFile, err)
			validConfig = false
		}
		viper.OnConfigChange(updateClientVersion)
		viper.WatchConfig()
	}
}

func updateClientVersion(in fsnotify.Event) {
	newVersion := viper.GetString("clientVersion")
	err := validateVersion(newVersion)
	if err != nil {
		panic(err)
	}
	setClientVersion(newVersion)
}

func setClientVersion(version string) {
	clientVersionLock.Lock()
	clientVersion = version
	clientVersionLock.Unlock()
}

func validateVersion(versionString string) error {
	// If a version string has more than 2 dots in it, anything after the first
	// 2 dots is considered to be part of the patch version
	versions := strings.SplitN(versionString, ".", 3)
	if len(versions) != 3 {
		return errors.New("Client version string must contain a major, minor, and patch version separated by \".\"")
	}
	_, err := strconv.Atoi(versions[0])
	if err != nil {
		return errors.New("Major client version couldn't be parsed as integer")
	}
	_, err = strconv.Atoi(versions[1])
	if err != nil {
		return errors.New("Minor client version couldn't be parsed as integer")
	}
	return nil
}

// initLog initializes logging thresholds and the log path.
func initLog() {
	if viper.Get("logPath") != nil {
		vipLogLevel := viper.GetUint("logLevel")

		// Check the level of logs to display
		if vipLogLevel > 1 {
			// Set the GRPC log level
			err := os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "info")
			if err != nil {
				jww.ERROR.Printf("Could not set GRPC_GO_LOG_SEVERITY_LEVEL: %+v", err)
			}

			err = os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "99")
			if err != nil {
				jww.ERROR.Printf("Could not set GRPC_GO_LOG_VERBOSITY_LEVEL: %+v", err)
			}
			// Turn on trace logs
			jww.SetLogThreshold(jww.LevelTrace)
			jww.SetStdoutThreshold(jww.LevelTrace)
			mixmessages.TraceMode()
		} else if vipLogLevel == 1 {
			// Turn on debugging logs
			jww.SetLogThreshold(jww.LevelDebug)
			jww.SetStdoutThreshold(jww.LevelDebug)
			mixmessages.DebugMode()
		} else {
			// Turn on info logs
			jww.SetLogThreshold(jww.LevelInfo)
			jww.SetStdoutThreshold(jww.LevelInfo)
		}
		jww.SetLogThreshold(jww.LevelTrace)
		jww.SetStdoutThreshold(jww.LevelTrace)

		// Create log file, overwrites if existing
		logPath := viper.GetString("logPath")
		logFile, err := os.Create(logPath)
		if err != nil {
			jww.WARN.Println("Invalid or missing log path, default path used.")
		} else {
			jww.SetLogOutput(logFile)
		}
	}
}
