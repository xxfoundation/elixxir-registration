package certAuth

import "github.com/spacemonkeygo/openssl"




func Sign(cert openssl.Certificate, privKey openssl.PrivateKey) {
	cert.Sign()
}