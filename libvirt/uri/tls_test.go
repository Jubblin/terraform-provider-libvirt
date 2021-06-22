package uri

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"github.com/stretchr/testify/assert"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func createClientCerts(pkipath string, caCertLoc string, caKeyLoc string) error {
	chain, err := tls.LoadX509KeyPair(caCertLoc, caKeyLoc)
	if err != nil {
		return err
	}

	ca, err := x509.ParseCertificate(chain.Certificate[0])
	if err != nil {
		return err
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject: pkix.Name{
			Organization: []string{"Avocado"},
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:    x509.KeyUsageDigitalSignature,
	}
	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey

	clientCert, err := x509.CreateCertificate(rand.Reader, clientTemplate, ca, pub, chain.PrivateKey)

	clientCertLoc := filepath.Join(pkipath, "clientcert.pem")
	clientKeyLoc := filepath.Join(pkipath, "clientkey.pem")

	certOut, err := os.Create(clientCertLoc)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: clientCert})
	certOut.Close()

	keyOut, err := os.OpenFile(clientKeyLoc, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	err = keyOut.Close()
	if err != nil {
		return err
	}

	return nil
}

func createCACerts(pkipath string) error {
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(42),
		Subject: pkix.Name{
			Organization: []string{"Avocado"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		IsCA:      true,
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}

	priv, _ := rsa.GenerateKey(rand.Reader, 2048)
	pub := &priv.PublicKey
	ca, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, pub, priv)
	if err != nil {
		return err
	}

	caCertLoc := filepath.Join(pkipath, "cacert.pem")
	caKeyLoc := filepath.Join(pkipath, "cakey.pem")

	certOut, err := os.Create(caCertLoc)
	if err != nil {
		return err
	}
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: ca})
	err = certOut.Close()
	if err != nil {
		return err
	}

	keyOut, err := os.OpenFile(caKeyLoc, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}

	pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(priv)})
	err = keyOut.Close()
	if err != nil {
		return err
	}

	return createClientCerts(pkipath, caCertLoc, caKeyLoc)
}

func TestNonZeroString(t *testing.T) {
	assert.False(t, nonZero("0"))
	assert.False(t, nonZero(""))
	assert.False(t, nonZero("000"))
	assert.True(t, nonZero("1"))
	assert.True(t, nonZero("A1B"))
	assert.True(t, nonZero("0001"))
}

func TestTLSConfig(t *testing.T) {
	pkipath := t.TempDir()

	createCACerts(pkipath)

	u, err := Parse(fmt.Sprintf("xxx+tls://servername/?no_verify=1&pkipath=%s", pkipath))
	assert.NoError(t, err)

	tlsConfig, err := u.tlsConfig()
	assert.NoError(t, err)

	assert.NotNil(t, tlsConfig)
	assert.True(t, tlsConfig.InsecureSkipVerify)

	u, err = Parse(fmt.Sprintf("xxx+tls://servername/?pkipath=%s", pkipath))
	assert.NoError(t, err)

	tlsConfig, err = u.tlsConfig()
	assert.NoError(t, err)

	assert.NotNil(t, tlsConfig)
	assert.False(t, tlsConfig.InsecureSkipVerify)

}
