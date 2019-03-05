# registration

Library containing the Registration Server for adding new clients to cMix

## Example Configuration File

```yaml
# ==================================
# Registration Server Configuration
# ==================================

# Verbose logging
verbose: "true"
# Path to log file
logPath: "registration.log"

# The listening address of this registration server
registrationAddress: "0.0.0.0:11420"

# Database connection information
dbUsername: "cmix"
dbPassword: ""
dbName: "cmix_server"
dbAddress: ""

# === REQUIRED FOR ENABLING TLS ===
# Path to the registration server private key file
keyPath: ""
# Path to the registration server certificate file
certPath: ""
```
