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
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/registration/database"
	"os"
	"path"
	"sync"
)

type clientVersion struct {
	major int
	minor int
	patch string
}

var (
	cfgFile            string
	verbose            bool
	showVer            bool
	noTLS              bool
	RegistrationCodes  []string
	RegParams          Params
	desiredVersion     *clientVersion
	desiredVersionLock sync.RWMutex
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "registration",
	Short: "Runs a registration server for cMix",
	Long:  `This server provides registration functions on cMix`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if showVer {
			printVersion()
			return
		}

		if verbose {
			err := os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "info")
			if err != nil {
				jww.ERROR.Printf("Could not set GRPC_GO_LOG_SEVERITY_LEVEL: %+v", err)
			}

			err = os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "2")
			if err != nil {
				jww.ERROR.Printf("Could not set GRPC_GO_LOG_VERBOSITY_LEVEL: %+v", err)
			}
		}

		// Parse config file options
		certPath := viper.GetString("certPath")
		keyPath := viper.GetString("keyPath")
		address := fmt.Sprintf("0.0.0.0:%d", viper.GetInt("port"))
		ndfOutputPath := viper.GetString("ndfOutputPath")

		// Set up database connection
		database.PermissioningDb = database.NewDatabase(
			viper.GetString("dbUsername"),
			viper.GetString("dbPassword"),
			viper.GetString("dbName"),
			viper.GetString("dbAddress"),
		)

		// Populate Client registration codes into the database
		database.PopulateClientRegistrationCodes([]string{"AAAA"}, 100)

		// Populate Node registration codes into the database
		RegistrationCodes = viper.GetStringSlice("registrationCodes")
		database.PopulateNodeRegistrationCodes(RegistrationCodes)

		// Populate params
		RegParams = Params{
			Address:       address,
			CertPath:      certPath,
			KeyPath:       keyPath,
			NdfOutputPath: ndfOutputPath,
		}
		jww.INFO.Println("Starting Permissioning")
		jww.INFO.Println("Starting User Registration")
		// Start registration server
		impl := StartRegistration(RegParams)

		// Begin the thread which handles the completion registration
		go nodeRegistrationCompleter(impl)

		// Wait forever to prevent process from ending
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
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Show verbose logs for debugging")
	rootCmd.Flags().BoolVarP(&showVer, "version", "V", false,
		"Show version information")

	rootCmd.Flags().StringVarP(&cfgFile, "config", "c",
		"", "Sets a custom config file path")

	rootCmd.Flags().BoolVar(&noTLS, "noTLS", false,
		"Runs without TLS enabled")
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
	newVersion, err := parseClientVersion(viper.GetString("desiredVersion"))
	if err != nil {
		// Setting the wrong version is a misconfiguration
		// In this case, we'll panic to make sure the mistake gets fixed ASAP
		jww.ERROR.Panic(err)
	}
	setDesiredVersion(newVersion)
}

// initLog initializes logging thresholds and the log path.
func initLog() {
	if viper.Get("logPath") != nil {
		// If verbose flag set then log more info for debugging
		if verbose || viper.GetBool("verbose") {
			jww.SetLogThreshold(jww.LevelDebug)
			jww.SetStdoutThreshold(jww.LevelDebug)
		} else {
			jww.SetLogThreshold(jww.LevelInfo)
			jww.SetStdoutThreshold(jww.LevelInfo)
		}
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
