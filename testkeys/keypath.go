package testkeys

import (
	"path/filepath"
	"runtime"
)

//The previously signed certificate in testkeys was generated using the following commands
//openssl x509 -req -days 360 -in <CSR> -CA <CA-CERT> -CAkey <CA-KEY> -CAcreateserial -out alreadySigned.crt -sha256
//The inputs (CA cert/key & CSR) were generated unrelated to the ones in testkeys (ie the following was run twice)
/*		CA TLS keypair generation		*/
//openssl req -new -newkey rsa:4096 -x509 -sha256 -days 365 \
//-nodes -out <CA-CERT> -keyout <CA-KEY> -subj <CA-SUBJ>

//where one output was put in test keys as the testing environment, and one generated a 'mysteriously' signed cert from
//a root ca cert/key pair that is not known (could be revoked or malicious)

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

func GetCACertPath() string {
	return filepath.Join(getDirForFile(), "gateway.cmix.rip.crt")
}

func GetCAKeyPath() string {
	return filepath.Join(getDirForFile(), "gateway.cmix.rip.key")
}

//Signed by a certificate that is not currently used by the CA (for testing)
func GetCertPath_PreviouslySigned() string {
	return filepath.Join(getDirForFile(), "cmix-alreadySigned.crt")
}

func GetDSAKeyPath() string {
	return filepath.Join(getDirForFile(), "dsaKey.pem")
}

func GetNDFPath() string {
	return filepath.Join(getDirForFile(), "ndf.json")
}