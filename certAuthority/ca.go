package certAuthority

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

//Sign takes in 3 files: one from the client (to be signed) and 2 from us, a cert and a private key
//It signs the certificate signing request (CSR) with the root CA keypair
//It returns the signed certificate and the root certificate so the requester can verify
func Sign(clientCert *x509.Certificate, caCert *x509.Certificate, caPrivKey interface{}) (string, error) {
	//Load certs and keys
	//Check that loadPrivateKey returned an expected interface
	switch caPrivKey.(type) {
	case *rsa.PrivateKey:
	case *ecdsa.PrivateKey:
	default:
		err := errors.New("Not an expected key type")
		return "", err
	}

	//Create a template certificate to be used in the signing of the clients CSR

	//Sign the certificate using the caCert as the parent certificate. This takes a generic interface for the public
	//and private key given as the last 2 args
	clientSignedCertBytes, err := x509.CreateCertificate(rand.Reader, clientCert, caCert, clientCert.PublicKey, caPrivKey)
	if err != nil {
		return "", err
	}

	//Create a block from the clientSignedCert
	pemBlock := &pem.Block{Type: "CERTIFICATE", Bytes: clientSignedCertBytes}
	//err = pem.Encode(_, &pem.Block{Type: "CERTIFICATE", Bytes: clientSignedCert})

	//encode the pem block, and then convert it into a string for return
	clientSignedCert := string(pem.EncodeToMemory(pemBlock))
	return clientSignedCert, err

}
