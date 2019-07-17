package testkeys

import (
	"path/filepath"
	"runtime"
)

func getDirForFile() string {
	// Get the filename we're in
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Dir(currentFile)
}

// These functions are used to cover TLS connection code in tests
func GetNodeCertPath() string {
	return filepath.Join(getDirForFile(), "cmix.rip.crt")
}

func GetNodeKeyPath() string {
	return filepath.Join(getDirForFile(), "cmix.rip.key")
}

func GetNodeCSRPath() string {
	return filepath.Join(getDirForFile(), "cmix.rip.csr")
}

func GetGatewayCertPath() string {
	return filepath.Join(getDirForFile(), "gateway.cmix.rip.crt")
}

func GetGatewayKeyPath() string {
	return filepath.Join(getDirForFile(), "gateway.cmix.rip.key")
}

//Signed by a certificate that is not currently used by the CA (for testing)
func GetCertPath_PreviouslySignature() string {
	return filepath.Join(getDirForFile(), "cmix-alreadySigned.crt")
}

//A certificate signed by the certificate authority
func GetNodeCertPath_KnownSignature() string {
	return filepath.Join(getDirForFile(), "cmix-signed.crt")
}
