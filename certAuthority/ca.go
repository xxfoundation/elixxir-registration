package certAuthority

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	jww "github.com/spf13/jwalterweatherman"
	"golang.org/x/tools/go/ssa/interp/testdata/src/errors"
	"math/big"
	"time"
)

//Sign takes in 3 files: one from the client (to be signed) and 2 from us, a cert and a private key
//It signs the certificate signing request (CSR) with the root CA keypair
//It returns the signed certificate and the root certificate so the requester can verify
func Sign(clientCSR *x509.CertificateRequest, caCert *x509.Certificate, caPrivKey interface{}) (string, error) {
	//Load certs and keys
	//Check that loadPrivateKey returned an expected interface
	switch caPrivKey.(type) {
	case *rsa.PrivateKey:
	default:
		jww.ERROR.Println("Not an expected key type")
		err := errors.New("Not an expected key type (expected rsa")
		return "", err
	}

	//Make sure that the csr is valid
	err := clientCSR.CheckSignature()
	if err != nil {
		jww.ERROR.Println(err.Error())
		return "", err

	}

	//Create a template certificate to be used in the signing of the clients CSR
	clientCertTemplate := CreateCertTemplate(clientCSR)

	//Sign the certificate using the caCert as the parent certificate. This takes a generic interface for the public
	//and private key given as the last 2 args
	clientSignedCertBytes, err := x509.CreateCertificate(rand.Reader, clientCertTemplate, caCert, clientCertTemplate.PublicKey, caPrivKey)
	if err != nil {
		jww.ERROR.Printf(err.Error())
		return "", err
	}

	//Create a block from the clientSignedCert
	pemBlock := &pem.Block{Type: "CERTIFICATE", Bytes: clientSignedCertBytes}
	//err = pem.Encode(_, &pem.Block{Type: "CERTIFICATE", Bytes: clientSignedCert})

	//encode the pem block, and then convert it into a string for return
	clientSignedCert := string(pem.EncodeToMemory(pemBlock))
	return clientSignedCert, err

}

//createCertTemplate returns a template to be used when creating a signed certificate for the client
func CreateCertTemplate(csr *x509.CertificateRequest) *x509.Certificate {
	// Maybe do something like this? Thoughts??
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
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
}
