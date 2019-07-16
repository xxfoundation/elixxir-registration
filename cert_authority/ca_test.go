package cert_authority

import (
	"gitlab.com/elixxir/registration/testkeys"
	"testing"
)

func TestSign(t *testing.T) {

}

//put this in the ca.go file if it turns out to be more involved
func TestSign_VerifySignatureSuccess(t *testing.T) {

}

//Check that an already signed cert does not pass
func TestSign_VerifySignatureFailure(t *testing.T)  {

	alreadySignedCert := loadCertificate(testkeys.GetCSR_AlreadySigned())
	CACert := loadCertificate(testkeys.GetGatewayCertPath())

	err := alreadySignedCert.CheckSignatureFrom(CACert)

	if err == nil {
		t.Errorf("Failed to detect a certificate not signed by the root CA")
	}
}

//Test all the file opening things? Almost certainly a waste of time *shrugs*??
func Test_LoadCert(t *testing.T) {

}

func TestLoadCSR(t *testing.T) {

}

func TestLoadPrivKey(t *testing.T) {

}

/*
func TestSign_FileIsValidCert(t *testing.T) {

}
*/
