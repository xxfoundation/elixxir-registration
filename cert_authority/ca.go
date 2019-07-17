package cert_authority

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	jww "github.com/spf13/jwalterweatherman"
	"math/big"
	"os"
	"time"
)

//Sign takes in 3 files: one from the client (to be signed) and 2 from us, a cert and a private key
//It signs the certificate signing request (CSR) with the root CA keypair
//It returns the signed certificate and the root certificate so the requester can verify
func Sign(clientCSR *x509.CertificateRequest, caCert *x509.Certificate, caPrivKey interface{}) ([]byte, *x509.Certificate) {
	//Load certs and keys
	//Check that loadPrivateKey returned an expected interface
	switch caPrivKey.(type) {
	case *ecdsa.PrivateKey:
	case *rsa.PrivateKey:
	default:
		jww.ERROR.Println("Not an expected key type")
	}

	//Make sure that the csr is valid
	err := clientCSR.CheckSignature()
	if err != nil {
		jww.ERROR.Panicf(err.Error())
	}

	//Create a template certificate to be used in the signing of the clients CSR
	clientCertTemplate := createCertTemplate(clientCSR)

	//Sign the certificate using the caCert as the parent certificate. This takes a generic interface for the public
	//and private key given as the last 2 args
	clientSignedCert, err := x509.CreateCertificate(rand.Reader, clientCertTemplate, caCert, clientCertTemplate.PublicKey, caPrivKey)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	//Question: return the raw, or just create a file (ie use writeToFile)

	return clientSignedCert, caCert

}

//Call to write the passed in cert to a file. Maybe we don't need this in the CA specifically?
//Either do this here or in the sign.
func writeToFile(signedCert []byte, filepath string) {
	clientCRTFile, err := os.Create(filepath)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	err = pem.Encode(clientCRTFile, &pem.Block{Type: "CERTIFICATE", Bytes: signedCert})
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	err = clientCRTFile.Close()
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
}

func createCertTemplate(csr *x509.CertificateRequest) *x509.Certificate {
	// Maybe do something like this?
	//serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	// use our brand new and shiny rng?
	//serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	return &x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,

		PublicKey:          csr.PublicKey,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		SerialNumber:       big.NewInt(2), //TODO probs need to edit this

		Issuer:    csr.Subject,
		NotBefore: time.Now(),
		//TODO figure out when client certs should expire
		// Thoughts on this reviewer?
		NotAfter: time.Now().Add(24 * time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature,
		// Use the below
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
}
