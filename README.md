# permissioning

Library containing the Permissioning Server for adding new clients and nodes to 
cMix

## Example Configuration File

```yaml
# ==================================
# Permissioning Server Configuration
# ==================================

# Log message level
logLevel: 1

# Verbose logging
verbose: "true"
# Path to log file
logPath: "registration.log"
# Path to the node topology permissioning info
ndfOutputPath: "ndf.json"
# Minimum number of nodes to begin running rounds. this differs from the number of members 
# in a team because some scheduling algorithms may require multiple teams worth of nodes at minimum
minimumNodes: 3

# UDB ID
udbID: 1

# Public address, used in NDF it gives to client
publicAdress: "0.0.0.0:11420"

# The listening port of this  server
port: 11420

# Database connection information
dbUsername: "cmix"
dbPassword: ""
dbName: "cmix_server"
dbAddress: ""

# Path to JSON file with list of Node registration codes (in order of network 
# placement)
RegCodesFilePath: "regCodes.json"

# List of client codes to be added to the database (for testing)
clientRegCodes:
  - "AAAA"
  - "BBBB"
  - "CCCC"
    

# Client version (will allow all versions with major version 0)
clientVersion: "0.0.0"

# === REQUIRED FOR ENABLING TLS ===
# Path to the permissioning server private key file
keyPath: ""
# Path to the permissioning server certificate file
certPath: ""

# E2E/CMIX Primes
groups:
  cmix:
    prime: "${cmix_prime}"
    generator: "${cmix_generator}"
  e2e:
    prime: "${e2e_prime}"
    generator: "${e2e_generator}"

# Selection of scheduling algorithm to use. Options are:
#   simple - Schedules multiple teams to maximize performance, does not randomly re-arrange teams, if only a single
#            only scheduling a single team, will use numerical ordering data for AlphaNet
#   secure - Schedules new teams randomly, has appropriate buffers to ensure
# unpredictability, designed for BetaNet
schedulingAlgorithm: "single"

# Path to file with config for scheduling algorithm within the user directory 
schedulingConfigPath: "schedulingConfig.json"
```

