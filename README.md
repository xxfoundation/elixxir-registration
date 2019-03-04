# registration

Library containing the Registration Server for adding new clients to cMix

## Example Configuration File

```yaml
# ==================================
# Registration Server Configuration
# ==================================

# The listening address of this registration server
registrationAddress: "0.0.0.0:11420"

# === REQUIRED FOR ENABLING TLS ===
# Path to the registration server private key file
keyPath: ""
# Path to the registration server certificate file
certPath: ""
```
