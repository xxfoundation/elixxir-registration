package certAuthority

import (
	"crypto/x509"
	"encoding/pem"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/crypto/tls"
	"gitlab.com/elixxir/primitives/utils"
	"gitlab.com/elixxir/registration/testkeys"
	"testing"
)

//Load files from filepaths that exist for testing purposes
func loadCertificate(file string) *x509.Certificate {
	pemEncodedBlock, err := utils.ReadFile(file)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	certDecoded, _ := pem.Decode(pemEncodedBlock)
	if certDecoded == nil {
		jww.ERROR.Printf("Decoding PEM Failed For %v", file)
	}

	cert, err := x509.ParseCertificate(certDecoded.Bytes)

	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	return cert

}

//loadPrivKey takes the file given and returns a private key of type ecdsa or rsa
func loadPrivKey(file string) interface{} {
	pemEncodedBlock, err := utils.ReadFile(file)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	certDecoded, _ := pem.Decode(pemEncodedBlock)
	if certDecoded == nil {
		jww.ERROR.Printf("Decoding PEM Failed For %v", file)
	}
	//Openssl creates pkcs8 keys by default as of openSSL 1.0.0 need to be able to handle both
	privateKey, err := x509.ParsePKCS8PrivateKey(certDecoded.Bytes)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	return privateKey
}

func ConvertToASNBytes(pemString string) *pem.Block {
	decodedBytes, _ := pem.Decode([]byte(pemString))

	return decodedBytes

}

//Test the checksign is implemented correctly in sign
func TestSign_CheckSignature(t *testing.T) {
	//Load files
	clientCert := loadCertificate(testkeys.GetNodeCertPath())
	caCert := loadCertificate(testkeys.GetCACertPath())
	pemKey, _ := utils.ReadFile(testkeys.GetCAKeyPath())
	caPrivKey, _ := tls.LoadRSAPrivateKey(string(pemKey))

	signedCertString, _ := Sign(clientCert, caCert, caPrivKey)
	signedCertBytes := ConvertToASNBytes(signedCertString)
	signedCert, err := x509.ParseCertificate(signedCertBytes.Bytes)
	if err != nil {
		t.Error(err)
	}
	err = signedCert.CheckSignatureFrom(caCert)
	if err != nil {
		t.Error("Certificate signature not constructed properly")
	}

}

//Check that an already signed cert does not pass
func TestSign_VerifySignatureFailure(t *testing.T) {

	alreadySignedCert := loadCertificate(testkeys.GetCertPath_PreviouslySigned())
	CACert := loadCertificate(testkeys.GetCACertPath())
	err := alreadySignedCert.CheckSignatureFrom(CACert)

	if err == nil {
		t.Errorf("Failed to detect a certificate not signed by the root CA")
	}
}

//Check that a private key with dsa is not accepted
func TestDSAKeyCert(t *testing.T) {
	clientCert := loadCertificate(testkeys.GetNodeCertPath())
	caCert := loadCertificate(testkeys.GetCACertPath())
	caPrivKey := loadPrivKey(testkeys.GetDSAKeyPath())

	_, err := Sign(clientCert, caCert, caPrivKey)
	if err == nil {
		t.Error("Failed to detect a DSA private key")
	}
}
