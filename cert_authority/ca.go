package cert_authority

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	jww "github.com/spf13/jwalterweatherman"
	"io/ioutil"
	"math/big"
	"os"
	"time"
)

//Take in 3 files: one from the client (to be signed) and 2 from us, a cert and a private key
func Sign(clientCSRFile, CACertFile, caPrivFile string) []byte {
	//Load certs and keys
	clientCSR := loadCertificateRequest(clientCSRFile)
	fmt.Println("loaded CSR")
	caCert := loadCertificate(CACertFile)
	fmt.Println("loaded CA cert")
	caPrivKey := loadPrivKey(caPrivFile)
	fmt.Println("loaded CA key")
	//
	//if caPrivKey
	//Make sure that the csr has not already been signed
	err := clientCSR.CheckSignature()
	if err != nil {
		jww.ERROR.Panicf(err.Error())
	}

	//Create a template certificate to be used in the signing of the clients CSR
	clientCertTemplate := createCertTemplate(clientCSR)
	fmt.Println(clientCSR.PublicKey)
	fmt.Println(clientCSR.PublicKeyAlgorithm)
	//Sign the certificate using the caCert as the parent certificate
	clientSignedCert, err := x509.CreateCertificate(rand.Reader, clientCertTemplate, caCert, clientCSR.PublicKey, caPrivKey)
	fmt.Println("pre err")
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	fmt.Println("post err")
	//return the raw, or just create a file
	//for testing purposes we could just return
	// wouldn't necesarily incorrect to store them in files
	// question would be is this a security flaw? Do root CAs keep all signatures.
	//TODO research whether CAs keep all signed certs locally (my guess is no)

	// Or we could do this, thoughts?
	/*
		clientCRTFile, err := os.Create("cert/client.crt") //name could be customized to "cert/" + nodeIDFromArgFileName + ".crt"
		err := pem.Encode(clientCRTFile, &pem.Block{Type:"CERTIFICATE", Bytes:clientSignedCert})
		if err != nil {
			jww.ERROR.Printf(err.Error())
		}

		clientCRTFile.Close()
	*/
	return clientSignedCert

}

//Call to write the passed in cert to a file. Maybe we don't need this in the CA specifically?
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
	return &x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,

		PublicKey:          csr.PublicKey,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		SerialNumber:big.NewInt(2),//TODO probs need to edit this
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

//Maybe simplify sign, move these to tests? Thoughts?
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
	fmt.Println(certDecoded.Headers)
	cert, err := x509.ParseCertificateRequest(certDecoded.Bytes)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	return cert
}
//hacked by making it return an interface
//TODO make sure it returns a ecdsa.private key (ie that it is pkcs8)
func loadPrivKey(file string) interface{} {
	pemEncodedBlock, err := ioutil.ReadFile(file)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	fmt.Println(pemEncodedBlock)
	certDecoded, _ := pem.Decode(pemEncodedBlock)
	if certDecoded == nil {
		jww.ERROR.Printf("Decoding PEM Failed For %v", file)
	}
	fmt.Println(certDecoded.Bytes)

	//Openssl creates pkcs8 by default now...
	privateKey, err :=  x509.ParsePKCS8PrivateKey(certDecoded.Bytes)

	if err != nil {
		jww.ERROR.Printf(err.Error())
	}
	return privateKey
}
