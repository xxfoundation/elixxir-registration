# permissioning

Library containing the Permissioning Server for adding new clients and nodes to 
cMix

## Example Configuration File

```yaml
# ==================================
# Permissioning Server Configuration
# ==================================

# Log message level (0 = info, 1 = debug, >1 = trace)
logLevel: 1

# Path to log file
logPath: "registration.log"

# Path to the node topology permissioning info
ndfOutputPath: "ndf.json"

# Path to JSON containing list of IDs exempt from rate limiting
whitelistedIdsPath: "whitelistedIds.json"

# Path to JSON containing list of IP addresses exempt from rate limiting
whitelistedIpAddressesPath: "whitelistedIpAddresses.json"

# Minimum number of nodes to begin running rounds. This differs from the number
# of members in a team because some scheduling algorithms may require multiple
# teams worth of nodes at minimum.
minimumNodes: 3

# "Location of the user discovery contact file.
udContactPath: "udContact.bin"

# Path to UDB cert file
udbCertPath: "udb.crt"

# Address for UDB
udbAddress: "1.2.3.4:11420"

# Public address, used in NDF it gives to nodes
publicAddress: "0.0.0.0:11420"

# The listening port of this server
port: 11420

# Public address used in NDF to give to client
registrationAddress: "5.6.7.8:11420"

# The minimum version required of gateways to connect
minGatewayVersion: "0.0.0"

# The minimum version required of servers to connect
minServerVersion:  "0.0.0"

# The minimum version required of clients to connect
minClientVersion: "0.0.0"

# Disable pruning of NDF for offline nodes
# if set to false, network will sleep for five minutes on start
disableNDFPruning: true

# disables the rejection of nodes and gateways with internal 
# or reserved IPs. For use within local environment or integration testing. 
permissiveIPChecking: false


# Database connection information
dbUsername: "cmix"
dbPassword: ""
dbName: "cmix_server"
dbAddress: ""

# Path to JSON file with list of Node registration codes (in order of network 
# placement)
regCodesFilePath: "regCodes.json"

# The duration between polling the disabled Node list for updates (Default 1m)
disabledNodesPollDuration: 1m

# Path to the text file with a list of IDs of disabled Nodes. If no path is,
# supplied, then the disabled Node list polling never starts.
disabledNodesPath: "disabledNodes.txt"

# === REQUIRED FOR ENABLING TLS ===
# Path to the permissioning server private key file
keyPath: ""
# Path to the permissioning server certificate file
certPath: ""

# Time interval (in seconds) between committing Node statistics to storage
nodeMetricInterval: 180

# Time interval (in minutes) in which the database is checked for banned nodes
BanTrackerInterval: "3"

# E2E/CMIX Primes
groups:
  cmix:
    prime: "${cmix_prime}"
    generator: "${cmix_generator}"
  e2e:
    prime: "${e2e_prime}"
    generator: "${e2e_generator}"

# Path to file with config for scheduling algorithm within the user directory 
schedulingConfigPath: "Scheduling_Simple_NonRandom.json"

# Time that the registration server waits before timing out while killing the
# round scheduling thread
schedulingKillTimeout: 10s
# Time the registration waits for rounds to close out and stop (optional)
closeTimeout: 60s

# Address of the notification server
nsAddress: ""
# Path to certificate for the notification server
nsCertPath: ""

# The initial size of the address space used for ephemeral IDs (Default: 5)
addressSpace: 32

# The interval between checks of storage for address space size updates.
# (Default 5m)
addressSpaceSizeUpdateInterval: 5m

# Toggles use of only active nodes in node metric tracker
onlyScheduleActive: false

# Toggles blockchain integration functionality
enableBlockchain: false

# A MaxMind GeoLite2 database file to lookup IPs against for geobinning
geoIPDBFile: "/GeoLite2-City.mmdb"


# For testing, use the sequence as the country code. Do not use the geobinning database
disableGeoBinning: false

# For testing, do not exclude node or gateway IPs which are local to the machine
allowLocalIPs: false

# Pulls geobin information from the blockchain instead of the hardcoded info
blockchainGeoBinning: false

# How long offline nodes remain in the NDF. If a node is offline past this duration
# the node is pruned from the NDF. Expects duration in"h". (Defaults to 1 week (168 hours)
pruneRetentionLimit: "168h"
```

### SchedulingConfig template:

Note: All times in MS

```json
{
  "TeamSize": 3,
  "BatchSize": 64,
  "MinimumDelay": 60,
  "RealtimeDelay": 3000,
  "Threshold": 10,
  "NodeCleanUpInterval": 180000,  
  "Secure": true,
  "PrecomputationTimeout": 30000,
  "RealtimeTimeout": 15000,
  "ResourceQueueTimeout": 180000,
  "DebugTrackRounds": true
}
```

### RegCodes Template
```json
[{"RegCode": "qpol", "Order": "0"},
{"RegCode": "yiiq", "Order": "1"},
{"RegCode": "vydz", "Order": "2"},
{"RegCode": "gwxs", "Order": "3"},
{"RegCode": "nahv", "Order": "4"},
{"RegCode": "plmd", "Order": "5"}]
```
