package certAuth

import (
	"github.com/spacemonkeygo/openssl"
)

//Take in two files: one from the client (to be signed) and one from us, probably a private key
//
func Sign(clientCertFile, CA_keyFile []byte) {

	openssl.LoadCertificateFromPEM(clientCertFile)

	cert.Sign()
}
