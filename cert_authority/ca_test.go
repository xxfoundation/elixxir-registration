package cert_authority

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	jww "github.com/spf13/jwalterweatherman"
	"gitlab.com/elixxir/registration/testkeys"
	"io/ioutil"
	"testing"
)

//Load files from filepaths that exist for testing purposes
func loadCertificate(file string) *x509.Certificate {
	pemEncodedBlock, err := ioutil.ReadFile(file)
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

func loadCertificateRequest(file string) *x509.CertificateRequest {
	pemEncodedBlock, err := ioutil.ReadFile(file)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	certDecoded, _ := pem.Decode(pemEncodedBlock)
	if certDecoded == nil {
		jww.ERROR.Printf("Decoding PEM Failed For %v", file)
	}
	cert, err := x509.ParseCertificateRequest(certDecoded.Bytes)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	return cert
}

//loadPrivKey takes the file given and returns a private key of type ecdsa or rsa
func loadPrivKey(file string) interface{} {
	pemEncodedBlock, err := ioutil.ReadFile(file)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	certDecoded, _ := pem.Decode(pemEncodedBlock)
	if certDecoded == nil {
		jww.ERROR.Printf("Decoding PEM Failed For %v", file)
	}

	//Openssl creates pkcs8 keys by default as of openSSL 1.0.0
	privateKey, err := x509.ParsePKCS8PrivateKey(certDecoded.Bytes)

	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	return privateKey
}

func getKnownSignature() *pem.Block {
	fileIo := []byte(`-----BEGIN CERTIFICATE-----
MIIFVjCCAz6gAwIBAgIBAjANBgkqhkiG9w0BAQsFADCBkjELMAkGA1UEBhMCVVMx
CzAJBgNVBAgMAkNBMRIwEAYDVQQHDAlDbGFyZW1vbnQxEDAOBgNVBAoMB0VsaXh4
aXIxFDASBgNVBAsMC0RldmVsb3BtZW50MRkwFwYDVQQDDBBnYXRld2F5LmNtaXgu
cmlwMR8wHQYJKoZIhvcNAQkBFhBhZG1pbkBlbGl4eGlyLmlvMB4XDTE5MDcxNjIz
MTkzMloXDTE5MDcxNzIzMTkzMlowADCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCC
AgoCggIBAMXOJ4lDDe2USdfy8uPTiIXbQ/e4k5nXwRuktTAsbqzjiFfqs8Z8WczJ
NTy9vHYlFJhxCTldPT9GDk5dHh8ZalYBnjoMtetW5jTcKH1KHY61LgWp3tFAMQRP
nnvHStpp+glNLHKDQZz+63UwdajbjlLWVE65yclqNj+P2h3ItIkpMIoVVgkqP69W
A5SbEXWm8OEYUx5UuYIsQUmxW+ftkSq6Enzz9uv+Z1bcGjUmnAhQ2rR8/hCV+41c
hGzIIZ6DvQClzvINK+dlaNObx55OzzCXy3n9RBtSmUEQTtTeKu+H1QeMKJh+s0/9
AnNU5QT8yqzxV03oItntS14WyjXfc0aWBanMkgD/D7MzbOaNoi34BTMNnusZ9PCt
Jd05ohYQptHwgcMqpVeWvG2dF4wCPb+C9apvKgGYism7LVJFghhtpCVGmcWf1QZN
WorSX/teHG+CFwEcLLkuUK+EvFQDt0IPqp+cGf/hc/YQdj6vMWB85ZAwodoviCYH
2zllkr56LWabv14IIDwhVxY3zIyEF0GtNe/R88zhB0aMPsGgwHU5qYVgDzUmk35+
O2Cn6y8w3rIRsW5tloNFhAelIEexK8JE5p0Kzv3scT2e4+GcKY4cqNIC6py0vkun
9P9VSKIHavRVgIJ7GoMX8BwfppoGfI/kqWbl5im+9jjbz3sMXzTdAgMBAAGjSDBG
MA4GA1UdDwEB/wQEAwIHgDATBgNVHSUEDDAKBggrBgEFBQcDAjAfBgNVHSMEGDAW
gBTBZRWWVIYwzrQ8wcgvcW8s38RUQzANBgkqhkiG9w0BAQsFAAOCAgEAKsqFlNQK
XPXaxGlLvmRCzz2LQidkektj6jV6AxNOhhhkY+0lHSf7JPN2JE9IqdH4YSuqSx/z
YK2t9NDv8zgUvkyL9m4IDPDja+8VFGw8wVUC4Oa6LZTGfzL7u6NZtqg2xNX1PXMs
t6y8x0Idnj6n16QFS8w+vQDxAmn4UOtDd4MOt7TUvrHsfNbF4+6QRW2EttjvLOHP
/y+JFi4LKYEvSq+FSImuzbNjc2MbclGK/QUR7LL99xa90JjEzKshIvbWs0hglufl
I05s7sxsoCvMXwDftj6onCP780+XERAjA9pXZAkaqsLxJ+eHiwntiYd+nS6edCb8
+CihW2kPjJ3YgdHa82jCkcT/qMZRKsel4csK67CqTtPgX3MnDV/gLvh2VclrZjab
rjsuxzGkrKI3RBouJShVxEVfS+4wxV7fsG73lLV0lehCp8ZVIlSkw9Y6wa5OciD2
yzj+M4m6C+bsxUV9Foi++ow+L8tJ35sP1v/OV5+GnI0VZPsvLmkk2eqCwgHECCqO
CnGgEV7kMbIJm53Ooy/nDxpXawRSlRjbAVnEmLAKy7iSYBOucx+BQ/3TnTQ9S7Ii
XObTGJ8pmDRq9vobLxvxZ6v5wle8nEef5HZW2ddcBQ/2cQdJNIgi7DJi86qj9gc1
8ScD4Dr1Gt4wnORAq0jHkl45CNICTCoplY0=
-----END CERTIFICATE-----`)
	ret, _ := pem.Decode(fileIo)
	return ret
}



//test repeatability by pulling the signed cert, resigning (they should be the same with the same csr, CACert
// and privKey
//Might be a hackey test since we're only comparing up to a certain length..Thoughts?
func TestSign_Consistency(t *testing.T) {
	//Load files
	clientCSR := loadCertificateRequest(testkeys.GetNodeCSRPath())
	caCert := loadCertificate(testkeys.GetCACertPath())
	caPrivKey := loadPrivKey(testkeys.GetCAKeyPath())

	expected := *getKnownSignature()

	observed, _ := Sign(clientCSR, caCert, caPrivKey)

	divison := 8
	//won't be exactly the same as some randomness is added
	if bytes.Compare(expected.Bytes[:len(expected.Bytes)/divison], observed[:len(observed)/divison]) != 0 {
		t.Error("Failed signature consistency")
	}
}

//Test the checksign is implemented correctly in sign
func TestSign_CheckSignature(t *testing.T) {
	//Load files
	clientCSR := loadCertificateRequest(testkeys.GetNodeCSRPath())
	caCert := loadCertificate(testkeys.GetCACertPath())
	caPrivKey := loadPrivKey(testkeys.GetCAKeyPath())

	signedCertBytes, _ := Sign(clientCSR, caCert, caPrivKey)
	signedCert, err := x509.ParseCertificate(signedCertBytes)
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


