package gonet

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"time"

	"github.com/pkg/errors"
)

func MustKeyPairWithPin() (pemCert, pemKey, skpiFingerprint []byte) {
	if pemCert, pemKey, skpiFingerprint, err := KeyPairWithPin(); err != nil {
		panic(errors.Wrap(err, "KeyPairWithPin"))
	} else {
		return pemCert, pemKey, skpiFingerprint
	}
}

// KeyPairWithPin returns PEM encoded Certificate and Key along with an SKPI fingerprint of the public key.
// refer https://blog.afoolishmanifesto.com/posts/golang-self-signed-and-pinned-certs/
func KeyPairWithPin() (pemCert, pemKey, skpiFingerprint []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "rsa.GenerateKey")
	}

	tpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "bjca"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(100, 0, 0),
		BasicConstraintsValid: true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
	}
	derCert, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "x509.CreateCertificate")
	}

	pemCertBuf := &bytes.Buffer{}
	if err := pem.Encode(pemCertBuf, &pem.Block{Type: "CERTIFICATE", Bytes: derCert}); err != nil {
		return nil, nil, nil, errors.Wrap(err, "pem.Encode")
	}

	pemCert = pemCertBuf.Bytes()

	pemKeyBuf := &bytes.Buffer{}
	err = pem.Encode(pemKeyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privateKey)})
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "pem.Encode")
	}
	pemKey = pemKeyBuf.Bytes()

	cert, err := x509.ParseCertificate(derCert)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "x509.ParseCertificate")
	}

	pubDER, err := x509.MarshalPKIXPublicKey(cert.PublicKey.(*rsa.PublicKey))
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "x509.MarshalPKIXPublicKey")
	}
	sum := sha256.Sum256(pubDER)
	pin := make([]byte, base64.StdEncoding.EncodedLen(len(sum)))
	base64.StdEncoding.Encode(pin, sum[:])

	return pemCert, pemKey, pin, nil
}

func MustCreateServerTLSConfig(serverKeyFile, serverCertFile, clientCertFile string) *tls.Config {
	if c, e := CreateServerTLSConfig(serverKeyFile, serverCertFile, clientCertFile); e != nil {
		panic("failed to create CreateServerTLSConfig " + e.Error())
	} else {
		return c
	}
}

func CreateServerTLSConfig(serverKeyFile, serverCertFile, clientCertFile string) (*tls.Config, error) {
	if clientCertFile == "" {
		return CreateServerTLSConfigV1(serverKeyFile, serverCertFile)
	}

	return CreateServerTLSConfigV2(serverKeyFile, serverCertFile, clientCertFile)
}

func MustCreateClientTLSConfig(clientKeyFile, clientCertFile string) *tls.Config {
	if c, e := CreateClientTLSConfig(clientKeyFile, clientCertFile); e != nil {
		panic("failed to create CreateClientTLSConfig " + e.Error())
	} else {
		return c
	}
}
func CreateClientTLSConfig(clientKeyFile, clientCertFile string) (*tls.Config, error) {
	if clientKeyFile == "" || clientCertFile == "" {
		return CreateClientTLSConfigV1()
	}

	return CreateClientTLSConfigV2(clientKeyFile, clientCertFile)
}

// CreateServerTLSConfigV1 create net.Listener base on serverKeyFile, serverCertFile.
func CreateServerTLSConfigV1(serverKeyFile, serverCertFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, err
	}

	return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
}

// CreateServerTLSConfigV2 create net.Listener base on serverKeyFile, serverCertFile and clientCertFile.
func CreateServerTLSConfigV2(serverKeyFile, serverCertFile, clientCertFile string) (*tls.Config, error) {
	c, err := CreateServerTLSConfigV1(serverKeyFile, serverCertFile)
	if err != nil {
		return nil, err
	}

	c.ClientCAs = x509.NewCertPool()
	c.ClientAuth = tls.RequireAndVerifyClientCert

	if certBytes, err := ioutil.ReadFile(clientCertFile); err != nil {
		return nil, fmt.Errorf("failed to read clientCertFile %s, error %v", clientCertFile, err)
	} else if ok := c.ClientCAs.AppendCertsFromPEM(certBytes); !ok {
		return nil, fmt.Errorf("failed to parse root clientCertFile %s", clientCertFile)
	}

	return c, nil
}

func CreateClientTLSConfigV1() (*tls.Config, error) {
	// #nosec G402
	return &tls.Config{InsecureSkipVerify: true}, nil
}

func CreateClientTLSConfigV2(clientKeyFile, clientCertFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		return nil, err
	}

	c, _ := CreateClientTLSConfigV1()
	c.RootCAs = x509.NewCertPool()
	c.Certificates = []tls.Certificate{cert}

	if certBytes, err := ioutil.ReadFile(clientCertFile); err != nil {
		return nil, fmt.Errorf("failed to read clientCertFile %s, error %v", clientCertFile, err)
	} else if ok := c.RootCAs.AppendCertsFromPEM(certBytes); !ok {
		return nil, fmt.Errorf("failed to parse root clientCertFile %s", clientCertFile)
	}
	return c, nil
}
