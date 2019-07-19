////////////////////////////////////////////////////////////////////////////////
// Copyright © 2018 Privategrity Corporation                                   /
//                                                                             /
// All rights reserved.                                                        /
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger

package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/crypto/large"
	"gitlab.com/elixxir/crypto/signature"
	"gitlab.com/elixxir/registration/database"
	"io/ioutil"
	"os"
)

var cfgFile string
var verbose bool
var showVer bool
var RegistrationCodes []string
var dsaKeyPairPath string

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

		//get the DSA private key
		dsaKeyBytes, err := ioutil.ReadFile(dsaKeyPairPath)

		if err != nil {
			jww.FATAL.Panicf("Could not read dsa keys file: %v", err)
		}

		dsaKeys := DSAKeysJson{}

		err = json.Unmarshal(dsaKeyBytes, &dsaKeys)

		if err != nil {
			jww.FATAL.Panicf("Could not unmarshal dsa keys file: %v", err)
		}

		dsaPrivInt := large.NewIntFromString(dsaKeys.PrivateKeyHex, 16)
		dsaPubInt := large.NewIntFromString(dsaKeys.PublicKeyHex, 16)

		pubKey := signature.ReconstructPublicKey(dsaParams, dsaPubInt)
		privateKey = signature.ReconstructPrivateKey(pubKey, dsaPrivInt)

		// Parse config file options
		certPath := viper.GetString("certPath")
		keyPath := viper.GetString("keyPath")
		address := fmt.Sprintf("0.0.0.0:%d", viper.GetInt("port"))

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
		params := Params{
			Address:  address,
			CertPath: certPath,
			KeyPath:  keyPath,
		}

		// Start registration server
		StartRegistration(params)

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
	rootCmd.Flags().StringVarP(&dsaKeyPairPath, "keyPairOverride", "k",
		"", "Defined a DSA keypair to use instead of generating a new one")
	rootCmd.Flags().StringVarP(&cfgFile, "configPath", "c",
		"", "Sets a custom  config file path")
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
		viper.SetConfigFile(cfgFile)
		viper.AutomaticEnv() // read in environment variables that match

		// If a config file is found, read it in.
		if err := viper.ReadInConfig(); err != nil {
			jww.ERROR.Printf("Unable to parse config file (%s): %+v", cfgFile, err)
			validConfig = false
		}
	}
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

type DSAKeysJson struct {
	PrivateKeyHex string
	PublicKeyHex  string
}
