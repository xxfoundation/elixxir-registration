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
	"gitlab.com/elixxir/primitives/version"
	"gitlab.com/elixxir/registration/scheduling"
	"gitlab.com/elixxir/registration/storage"
	"gitlab.com/elixxir/registration/storage/node"
	"net"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
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
	disabledNodesPath    string

	// Duration between polls of the disabled Node list for updates.
	disabledNodesPollDuration time.Duration
)

// Default duration between polls of the disabled Node list for updates.
const defaultDisabledNodesPollDuration = time.Minute

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
		roundIdPath := viper.GetString("roundIdPath")
		updateIdPath := viper.GetString("updateIdPath")

		maxRegistrationAttempts := viper.GetUint64("maxRegistrationAttempts")
		if maxRegistrationAttempts == 0 {
			maxRegistrationAttempts = defaultMaxRegistrationAttempts
		}

		registrationCountDuration := viper.GetDuration("registrationCountDuration")
		if registrationCountDuration == 0 {
			registrationCountDuration = defaultRegistrationCountDuration
		}

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
			regCodeInfos, err := node.LoadInfo(RegCodesFilePath)
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

		disableGatewayPing := viper.GetBool("disableGatewayPing")

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
			schedulingKillTimeout:     schedulingKillTimeout,
			closeTimeout:              closeTimeout,
			udbId:                     udbId,
			minimumNodes:              viper.GetUint32("minimumNodes"),
			minGatewayVersion:         minGatewayVersion,
			minServerVersion:          minServerVersion,
			roundIdPath:               roundIdPath,
			updateIdPath:              updateIdPath,
			disableGatewayPing:        disableGatewayPing,
		}

		jww.INFO.Println("Starting Permissioning Server...")

		// Start registration server
		quitRegistrationCapacity := make(chan bool)
		impl, err := StartRegistration(RegParams, quitRegistrationCapacity)
		if err != nil {
			jww.FATAL.Panicf(err.Error())
		}

		err = impl.LoadAllRegisteredNodes()
		if err != nil {
			jww.FATAL.Panicf("Could not load all nodes from database: %+v", err)
		}

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

		// Determine how long between storing Node metrics
		nodeMetricInterval := time.Duration(
			viper.GetInt64("nodeMetricInterval")) * time.Second
		nodeTicker := time.NewTicker(nodeMetricInterval)

		// Run the Node metric tracker forever in another thread
		metricTrackerQuitChan := make(chan struct{})
		go func(quitChan chan struct{}) {
			jww.DEBUG.Printf("Beginning storage of node metrics every %+v...",
				nodeMetricInterval)
			for {
				// Store the metric start time
				startTime := time.Now()
				select {
				// Wait for the ticker to fire
				case <-nodeTicker.C:

					// Iterate over the Node States
					nodeStates := impl.State.GetNodeMap().GetNodeStates()
					for _, nodeState := range nodeStates {

						// Build the NodeMetric
						currentTime := time.Now()
						metric := &storage.NodeMetric{
							NodeId:    nodeState.GetID().Bytes(),
							StartTime: startTime,
							EndTime:   currentTime,
							NumPings:  nodeState.GetAndResetNumPolls(),
						}

						// Store the NodeMetric
						err := storage.PermissioningDb.InsertNodeMetric(metric)
						if err != nil {
							jww.FATAL.Panicf(
								"Unable to store node metric: %+v", err)
						}
					}
				}
			}
		}(metricTrackerQuitChan)

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

		roundCreationQuitChan := make(chan chan struct{})

		// Begin scheduling algorithm
		go func() {
			err = scheduling.Scheduler(SchedulingConfig, impl.State, roundCreationQuitChan)
			jww.FATAL.Panicf("Scheduling Algorithm exited: %s", err)
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

			// Try a non-blocking send for the registration capacity
			select {
			case quitRegistrationCapacity <- true:
			default:
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

			// Close connection to the database
			err = closeFunc()
			if err != nil {
				jww.ERROR.Printf("Error closing database: %+v", err)
			}
		}
		stopEverything := func() {
			stopOnce.Do(stopRounds)
			stopForKillOnce.Do(stopForKill)
		}
		ReceiveUSR2Signal(stopEverything)

		// Block forever on Signal Handler for safe program exit
		ReceiveExitSignal(func() int {
			stopEverything()
			if atomic.LoadUint32(impl.Stopped) == 1 {
				return 0
			} else {
				return -1
			}
		})

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

	rootCmd.Flags().StringP("close-timeout", "t", "60s",
		("Amount of time to wait for rounds to stop running after" +
			" receiving the SIGUSR1 and SIGTERM signals"))

	rootCmd.Flags().StringP("kill-timeout", "k", "60s",
		("Amount of time to wait for round creation to stop after" +
			" receiving the SIGUSR2 and SIGTERM signals"))

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
