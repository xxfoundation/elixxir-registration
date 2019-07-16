package certAuth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	jww "github.com/spf13/jwalterweatherman"
	"io/ioutil"
)

//Take in two files: one from the client (to be signed) and one from us, probably a private key
//
func Sign(clientCSRFile, CACertFile, caPrivFile string) {
	//Load certs and keys
	clientCSR := loadCertificateRequest(clientCSRFile)
	caCert := loadCertificate(CACertFile)
	caPrivKey := loadPrivKey(caPrivFile)

	//Make sure that the csr has not already been signed
	err := clientCSR.CheckSignature(); if err != nil {
		jww.ERROR.Panicf(err.Error())
	}



}

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

func loadPrivKey(file string) *rsa.PrivateKey {
	pemEncodedBlock, err := ioutil.ReadFile(file)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	certDecoded, _ := pem.Decode(pemEncodedBlock)
	if certDecoded == nil {
		jww.ERROR.Printf("Decoding PEM Failed For %v", file)
	}
	//TODO test which of these i have to do, the uncommented or the commented one
	//  i.e figure out if priv key has a password associated..
	//  it may not, but if it doesn't ask if it will in the future
	//der, err := x509.DecryptPEMBlock(certDecoded, []byte(""))
	//May have to decrypt
	privateKey, err := x509.ParsePKCS1PrivateKey(certDecoded.Bytes)
	if err != nil {
		jww.ERROR.Printf(err.Error())
	}

	return privateKey
}
