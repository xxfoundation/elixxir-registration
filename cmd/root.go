////////////////////////////////////////////////////////////////////////////////
// Copyright © 2022 xx foundation                                             //
//                                                                            //
// Use of this source code is governed by a license that can be found in the  //
// LICENSE file.                                                              //
////////////////////////////////////////////////////////////////////////////////

// Package cmd initializes the CLI and config parsers as well as the logger

package cmd

import (
	"errors"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/comms/mixmessages"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/elixxir/registration/scheduling"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"gitlab.com/xx_network/primitives/utils"
	"net"
	"os"
	"path"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"
)

var (
	cfgFile              string
	logLevel             uint // 0 = info, 1 = debug, >1 = trace
	noTLS                bool
	RegParams            Params
	disablePermissioning bool
	disabledNodesPath    string

	// Storage of registration codes from file so it can be loaded from disableRegCodes
	regCodeInfos    []node.Info
	disableRegCodes bool

	// Duration between polls of the disabled Node list for updates.
	disabledNodesPollDuration time.Duration
)

// Default duration between polls of the disabled Node list for updates.
const defaultDisabledNodesPollDuration = time.Minute
const defaultPruneRetention = 24 * 7 * time.Hour
const defaultMessageRetention = 24 * 7 * time.Hour

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "registration",
	Short: "Runs a registration server for cMix",
	Long:  `This server provides registration functions on cMix`,
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		profileOut := viper.GetString("profile-out")
		if profileOut != "" {
			cpuPath := profileOut + "-cpu"
			memPath := profileOut + "-mem"
			fileFlags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
			fileOpts := os.FileMode(0644)

			// Start CPU profiling
			cpuFile, err := os.OpenFile(cpuPath, fileFlags, fileOpts)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}
			err = pprof.StartCPUProfile(cpuFile)
			if err != nil {
				jww.FATAL.Panicf("%+v", err)
			}

			// Start memory profiling
			go func() {
				for {
					memFile, err := os.OpenFile(memPath, fileFlags, fileOpts)
					if err != nil {
						jww.FATAL.Panicf("%+v", err)
					}
					err = pprof.WriteHeapProfile(memFile)
					if err != nil {
						jww.FATAL.Panicf("%+v", err)
					}
					err = memFile.Close()
					if err != nil {
						jww.FATAL.Panicf("%+v", err)
					}
					time.Sleep(time.Minute)
				}
			}()
		}

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
		fullNdfOutputPath := viper.GetString("fullNdfOutputPath")
		signedPartialNdfOutputPath := viper.GetString("signedPartialNDFOutputPath")
		whitelistedIdsPath := viper.GetString("whitelistedIdsPath")
		whitelistedIpAddressesPath := viper.GetString("whitelistedIpAddressesPath")

		ipAddr := viper.GetString("publicAddress")
		// Get Notification Server address and cert Path
		nsCertPath := viper.GetString("nsCertPath")
		nsAddress := viper.GetString("nsAddress")
		publicAddress := fmt.Sprintf("%s:%d", ipAddr, viper.GetInt("port"))
		clientRegistration := viper.GetString("registrationAddress")
		// Set up database connection
		rawAddr := viper.GetString("dbAddress")

		var addr, port string
		if rawAddr != "" {
			addr, port, err = net.SplitHostPort(rawAddr)
			if err != nil {
				jww.FATAL.Panicf("Unable to get database port: %+v", err)
			}
		}

		var closeFunc func() error // Used for closing the database
		storage.PermissioningDb, closeFunc, err = storage.NewDatabase(
			viper.GetString("dbUsername"),
			viper.GetString("dbPassword"),
			viper.GetString("dbName"),
			addr,
			port,
		)
		if err != nil {
			jww.FATAL.Panicf("Unable to initialize storage: %+v", err)
		}

		// Populate Node registration codes into the database
		RegCodesFilePath := viper.GetString("regCodesFilePath")
		if RegCodesFilePath != "" {
			regCodeInfos, err = node.LoadInfo(RegCodesFilePath)
			if err != nil {
				jww.ERROR.Printf("Failed to load registration codes from the "+
					"file %s: %+v", RegCodesFilePath, err)
			} else {
				storage.PopulateNodeRegistrationCodes(regCodeInfos)
			}
		} else {
			jww.WARN.Printf("No registration code file found. This may be" +
				"normal in live deployments")
		}

		contactPath := viper.GetString("udContactPath")
		contactFile, err := utils.ReadFile(contactPath)
		if err != nil {
			jww.FATAL.Panicf("Failed to read contact file path at %q: %+v", contactPath, err)
		}

		// Get user discovery ID and DH public key from contact file
		udbId, udbDhPubKey, err := contact.ReadContactFromFile(contactFile)
		if err != nil {
			jww.FATAL.Panicf("Failed to read contact file path %q: %+v", contactPath, err)
		}

		// Get UDB cert path and address
		udbCertPath := viper.GetString("udbCertPath")
		udbAddress := viper.GetString("udbAddress")

		// load the scheduling params file as a string
		SchedulingConfigPath := viper.GetString("schedulingConfigPath")
		SchedulingConfig, err := utils.ReadFile(SchedulingConfigPath)
		if err != nil {
			jww.FATAL.Panicf("Could not load Scheduling Config file: %v", err)
		}

		// Parse version strings
		minGatewayVersionString := viper.GetString("minGatewayVersion")
		minGatewayVersion, err := version.ParseVersion(minGatewayVersionString)
		if err != nil {
			jww.FATAL.Panicf("Could not parse minGatewayVersion %#v: %+v",
				minGatewayVersionString, err)
		}

		minServerVersionString := viper.GetString("minServerVersion")
		minServerVersion, err := version.ParseVersion(minServerVersionString)
		if err != nil {
			jww.FATAL.Panicf("Could not parse minServerVersion %#v: %+v",
				minServerVersionString, err)
		}

		minClientVersionString := viper.GetString("minClientVersion")
		minClientVersion, err := version.ParseVersion(minClientVersionString)
		if err != nil {
			jww.FATAL.Panicf("Could not parse minClientVersion %#v: %+v",
				minClientVersionString, err)
		}

		// Get the amount of time to wait for scheduling to end
		// This should default to 10 seconds in StartRegistration if not set
		schedulingKillTimeout, err := time.ParseDuration(
			viper.GetString("schedulingKillTimeout"))
		if err != nil {
			jww.FATAL.Panicf("Could not parse duration: %+v", err)
		}

		// The amount of time to wait for rounds to stop running
		closeTimeout, err := time.ParseDuration(
			viper.GetString("closeTimeout"))
		if err != nil {
			jww.FATAL.Panicf("Could not parse duration: %+v", err)
		}

		viper.SetDefault("addressSpace", 5)
		viper.SetDefault("pruneRetentionLimit", defaultPruneRetention)

		viper.SetDefault("messageRetentionLimit", defaultMessageRetention)

		// Get rate limiting values
		capacity := viper.GetUint32("RateLimiting.Capacity")
		if capacity == 0 {
			capacity = 1
		}
		leakedTokens := viper.GetUint32("RateLimiting.LeakedTokens")
		if leakedTokens == 0 {
			leakedTokens = 1
		}
		leakedDurations := viper.GetUint64("RateLimiting.LeakDuration")
		if leakedTokens == 0 {
			leakedDurations = 2000
		}
		leakedDurations = leakedDurations * uint64(time.Millisecond)

		// Populate params
		RegParams = Params{
			Address:                    localAddress,
			CertPath:                   certPath,
			KeyPath:                    keyPath,
			FullNdfOutputPath:          fullNdfOutputPath,
			SignedPartialNdfOutputPath: signedPartialNdfOutputPath,
			WhitelistedIdsPath:         whitelistedIdsPath,
			WhitelistedIpAddressPath:   whitelistedIpAddressesPath,
			NsCertPath:                 nsCertPath,
			NsAddress:                  nsAddress,
			cmix:                       *cmix,
			e2e:                        *e2e,
			publicAddress:              publicAddress,
			clientRegistrationAddress:  clientRegistration,
			schedulingKillTimeout:      schedulingKillTimeout,
			closeTimeout:               closeTimeout,
			minimumNodes:               viper.GetUint32("minimumNodes"),
			udbId:                      udbId,
			udbDhPubKey:                udbDhPubKey,
			udbCertPath:                udbCertPath,
			udbAddress:                 udbAddress,
			minGatewayVersion:          minGatewayVersion,
			minServerVersion:           minServerVersion,
			minClientVersion:           minClientVersion,
			addressSpaceSize:           uint8(viper.GetUint("addressSpace")),
			allowLocalIPs:              viper.GetBool("allowLocalIPs"),
			disableGeoBinning:          viper.GetBool("disableGeoBinning"),
			blockchainGeoBinning:       viper.GetBool("blockchainGeoBinning"),
			onlyScheduleActive:         viper.GetBool("onlyScheduleActive"),
			enableBlockchain:           viper.GetBool("enableBlockchain"),

			disableNDFPruning:     viper.GetBool("disableNDFPruning"),
			geoIPDBFile:           viper.GetString("geoIPDBFile"),
			pruneRetentionLimit:   viper.GetDuration("pruneRetentionLimit"),
			messageRetentionLimit: viper.GetDuration("messageRetentionLimit"),
			versionLock:           sync.RWMutex{},

			// Rate limiting specs
			leakedCapacity: capacity,
			leakedTokens:   leakedTokens,
			leakedDuration: leakedDurations,
		}

		// Determine how long between storing Node metrics
		nodeMetricInterval := time.Duration(
			viper.GetInt64("nodeMetricInterval")) * time.Second

		jww.INFO.Println("Starting Permissioning Server...")
		jww.INFO.Printf("Params: %+v", RegParams)

		LoadAllRegNodes = true

		// Start registration server
		impl, err := StartRegistration(RegParams)
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}

		viper.OnConfigChange(impl.update)
		viper.WatchConfig()

		// Get disabled Nodes poll duration from config file or default to 1
		// minute if not set
		disabledNodesPollDuration = viper.GetDuration("disabledNodesPollDuration")
		if disabledNodesPollDuration == 0 {
			disabledNodesPollDuration = defaultDisabledNodesPollDuration
		}

		// Start routine to update disabled Nodes list
		disabledNodePollQuitChan := make(chan struct{})
		disabledNodesPath = viper.GetString("disabledNodesPath")
		if disabledNodesPath != "" {
			err = impl.State.CreateDisabledNodes(disabledNodesPath, disabledNodesPollDuration)
			if err != nil {
				jww.WARN.Printf("Error while parsing disabled Node list: %v", err)
			} else {
				go impl.State.StartPollDisabledNodes(disabledNodePollQuitChan)
			}
		} else {
			jww.DEBUG.Printf("No disabled Node list path provided. Skipping " +
				"disabled Node list polling.")
		}

		// Parse params JSON
		params := scheduling.ParseParams(SchedulingConfig)

		// Initialize param update if it is enabled
		if impl.params.enableBlockchain {
			go scheduling.UpdateParams(params, nodeMetricInterval)
		}

		impl.schedulingParams = params

		// Run the Node metric tracker forever in another thread
		metricTrackerQuitChan := make(chan struct{})
		go TrackNodeMetrics(impl, metricTrackerQuitChan, nodeMetricInterval)

		// Run address space updater until stopped
		viper.SetDefault("addressSpaceSizeUpdateInterval", 5*time.Minute)
		addressSpaceSizeUpdateInterval := viper.GetDuration("addressSpaceSizeUpdateInterval")
		addressSpaceTrackerQuitChan := make(chan struct{})
		go impl.TrackAddressSpaceSizeUpdates(addressSpaceSizeUpdateInterval,
			storage.PermissioningDb, addressSpaceTrackerQuitChan)

		// Determine how long between polling for banned nodes
		interval := viper.GetInt("BanTrackerInterval")
		ticker := time.NewTicker(time.Duration(interval) * time.Minute)

		// Run the independent node tracker in own go thread
		bannedNodeTrackerQuitChan := make(chan struct{})
		go func(quitChan chan struct{}) {
		nodeTrackerLoop:
			for {
				select {
				case <-ticker.C:
					// Keep track of banned nodes
					err = BannedNodeTracker(impl)
					if err != nil {
						jww.FATAL.Panicf("BannedNodeTracker failed: %v", err)
					}
				case <-quitChan:
					break nodeTrackerLoop
				}

			}
		}(bannedNodeTrackerQuitChan)

		jww.INFO.Printf("Waiting for for %v nodes to register so "+
			"rounds can start", RegParams.minimumNodes)

		<-impl.beginScheduling
		jww.INFO.Printf("Minimum number of nodes %v have registered, "+
			"beginning scheduling and round creation", RegParams.minimumNodes)
		err = impl.State.UpdateOutputNdf()
		if err != nil {
			jww.FATAL.Panicf("Failed to update output NDF with "+
				"registered nodes for scheduling: %+v", err)
		}

		roundCreationQuitChan := make(chan chan struct{})

		// Begin scheduling algorithm
		go func() {
			// Initialize scheduling
			err = scheduling.Scheduler(params, impl.State, roundCreationQuitChan)
			if err == nil {
				err = errors.New("")
			}
			jww.FATAL.Panicf("Scheduling Algorithm exited: %v", err)
		}()

		var stopOnce sync.Once
		// Set up signal handler for stopping round creation
		stopRounds := func() {
			k := make(chan struct{})
			roundCreationQuitChan <- k
			jww.INFO.Printf("Stopping round creation...")
			select {
			case <-k:
				jww.INFO.Printf("stopped!\n")
			case <-time.After(closeTimeout):
				jww.ERROR.Print("couldn't stop round creation!")
			}

			bannedNodeTrackerQuitChan <- struct{}{}

			// Prevent node updates after round creation stops
			atomic.StoreUint32(impl.Stopped, 1)
		}
		ReceiveUSR1Signal(func() { stopOnce.Do(stopRounds) })

		var stopForKillOnce sync.Once
		// Stops the long-running threads which are used for tracking node activity
		// You should only do this if you're killing permissioning outright,
		// as these threads are used to determine whether nodes get paid
		stopForKill := func() {
			jww.INFO.Printf("Stopping all other long-running threads")

			// Stop round metrics tracker
			metricTrackerQuitChan <- struct{}{}

			// Stop polling for disabled Nodes
			disabledNodePollQuitChan <- struct{}{}

			// Stop address space tracker
			addressSpaceTrackerQuitChan <- struct{}{}

			// Close GeoIP2 reader
			impl.geoIPDBStatus.ToStopped()
			err := impl.geoIPDB.Close()
			if err != nil {
				jww.ERROR.Printf("Error closing GeoIP2 database reader: %+v", err)
			}

			// Close connection to the database
			err = closeFunc()
			if err != nil {
				jww.ERROR.Printf("Error closing database: %+v", err)
			}
		}
		stopEverything := func() {
			if profileOut != "" {
				pprof.StopCPUProfile()
			}
			stopOnce.Do(stopRounds)
			stopForKillOnce.Do(stopForKill)
			impl.Comms.Shutdown()
		}
		ReceiveUSR2Signal(stopEverything)

		// Open Signal Handler for safe program exit
		stopCh := ReceiveExitSignal()

		// Block forever to prevent the program ending
		// Block until a signal is received, then call the function
		// provided
		select {
		case <-stopCh:
			jww.INFO.Printf(
				"Received Exit (SIGTERM or SIGINT) signal...\n")
			stopEverything()
			if atomic.LoadUint32(impl.Stopped) != 1 {
				os.Exit(-1)
			}
		}
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

	rootCmd.Flags().BoolVar(&disableRegCodes, "disableRegCodes", false,
		"Automatically provide registration codes to Nodes. (For testing only)")

	rootCmd.Flags().StringP("close-timeout", "t", "60s",
		"Amount of time to wait for rounds to stop running after"+
			" receiving the SIGUSR1 and SIGTERM signals")

	rootCmd.Flags().StringP("kill-timeout", "k", "60s",
		"Amount of time to wait for round creation to stop after"+
			" receiving the SIGUSR2 and SIGTERM signals")

	rootCmd.Flags().BoolVarP(&disablePermissioning, "disablePermissioning", "",
		false, "Disables registration server checking for ndf updates")

	err := viper.BindPFlag("closeTimeout",
		rootCmd.Flags().Lookup("close-timeout"))
	if err != nil {
		jww.FATAL.Panicf("could not bind flag: %+v", err)
	}

	err = viper.BindPFlag("schedulingKillTimeout",
		rootCmd.Flags().Lookup("kill-timeout"))
	if err != nil {
		jww.FATAL.Panicf("could not bind flag: %+v", err)
	}

	err = viper.BindPFlag("schedulingKillTimeout",
		rootCmd.Flags().Lookup("kill-timeout"))
	if err != nil {
		jww.FATAL.Panicf("could not bind flag: %+v", err)
	}

	rootCmd.Flags().String("udContactPath", "",
		"Location of the user discovery contact file.")

	err = viper.BindPFlag("udContactPath", rootCmd.Flags().Lookup("udContactPath"))
	if err != nil {
		jww.FATAL.Panicf("could not bind flag: %+v", err)
	}

	rootCmd.Flags().String("profile-out", "",
		"Enable profiling to this base file path")
	err = viper.BindPFlag("profile-out", rootCmd.Flags().Lookup("profile-out"))
	if err != nil {
		jww.FATAL.Panicf("could not bind flag: %+v", err)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Use default config location if none is passed
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
	}
}

func (m *RegistrationImpl) update(in fsnotify.Event) {
	m.updateVersions()
	m.updateRateLimiting()
	m.updateEarliestRound()

}

func (m *RegistrationImpl) updateEarliestRound() {
	msgRetention := viper.GetDuration("messageRetentionLimit")
	if msgRetention.Seconds() == 0 {
		msgRetention = defaultMessageRetention
	}

	m.params.messageRetentionLimitMux.Lock()
	m.params.messageRetentionLimit = msgRetention
	m.params.messageRetentionLimitMux.Unlock()

}

func (m *RegistrationImpl) updateRateLimiting() {
	// Get rate limiting values
	capacity := viper.GetUint32("RateLimiting.Capacity")
	if capacity == 0 {
		capacity = 1
	}
	leakedTokens := viper.GetUint32("RateLimiting.LeakedTokens")
	if leakedTokens == 0 {
		leakedTokens = 1
	}
	leakedDurations := viper.GetUint64("RateLimiting.LeakDuration")
	if leakedTokens == 0 {
		leakedDurations = 2000
	}
	leakedDurations = leakedDurations * uint64(time.Millisecond)

	m.State.InternalNdfLock.Lock()
	currentNdf := m.State.GetUnprunedNdf()
	currentNdf.RateLimits.Capacity = uint(capacity)
	currentNdf.RateLimits.LeakedTokens = uint(leakedTokens)
	currentNdf.RateLimits.LeakDuration = leakedDurations

	m.State.InternalNdfLock.Unlock()

}

func (m *RegistrationImpl) updateVersions() {
	// Parse version strings
	clientVersion := viper.GetString("minClientVersion")
	_, err := version.ParseVersion(clientVersion)
	if err != nil {
		jww.FATAL.Panicf("Attempted client version update is invalid: %v", err)
	}

	minGatewayVersionString := viper.GetString("minGatewayVersion")
	minGatewayVersion, err := version.ParseVersion(minGatewayVersionString)
	if err != nil {
		jww.FATAL.Panicf("Could not parse minGatewayVersion %#v: %+v",
			minGatewayVersionString, err)
	}

	minServerVersionString := viper.GetString("minServerVersion")
	minServerVersion, err := version.ParseVersion(minServerVersionString)
	if err != nil {
		jww.FATAL.Panicf("Could not parse minServerVersion %#v: %+v",
			minServerVersionString, err)
	}

	// Modify the client version
	m.State.InternalNdfLock.Lock()
	updateNDF := m.State.GetUnprunedNdf()
	jww.DEBUG.Printf("Updating client version from %s to %s", updateNDF.ClientVersion, clientVersion)
	updateNDF.ClientVersion = clientVersion
	m.State.UpdateInternalNdf(updateNDF)
	m.State.InternalNdfLock.Unlock()

	// Modify server and gateway versions
	m.params.versionLock.Lock()
	m.params.minGatewayVersion = minGatewayVersion
	m.params.minServerVersion = minServerVersion
	m.params.versionLock.Unlock()
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

		// Create log file, overwrites if existing
		logPath := viper.GetString("logPath")
		fullLogPath, _ := utils.ExpandPath(logPath)
		logFile, err := os.OpenFile(fullLogPath,
			os.O_CREATE|os.O_WRONLY|os.O_APPEND,
			0644)
		if err != nil {
			jww.WARN.Println("Invalid or missing log path, default path used.")
		} else {
			jww.SetLogOutput(logFile)
		}
	}
}
