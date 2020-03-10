# permissioning

Library containing the Permissioning Server for adding new clients and nodes to 
cMix

## Example Configuration File

```yaml
# ==================================
# Permissioning Server Configuration
# ==================================

# Verbose logging
verbose: "true"
# Path to log file
logPath: "registration.log"
# Path to the node topology permissioning info
ndfOutputPath: "ndf.json"
# Batch size of each round
batchSize: 8

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

# List of Node registration codes (in order of network placement)
registrationCodes:
  - "1"
  - "2"
  - "3"

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
```

