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

# === REQUIRED FOR ENABLING TLS ===
# Path to the permissioning server private key file
keyPath: ""
# Path to the permissioning server certificate file
certPath: ""
# Path to the node topology permissioning info
ndfOutputPath:
```

