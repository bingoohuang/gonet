package tlsconf

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/pkg/errors"
)

// NewHTTPSTestServer news a test https server.
func NewHTTPSTestServer(handler http.Handler, serverCertFile, serverKeyFile, clientRootCA string) *httptest.Server {
	ts := httptest.NewUnstartedServer(handler)
	ts.TLS = CreateServer(serverKeyFile, serverCertFile, clientRootCA)
	ts.StartTLS()

	return ts
}

// CreateServer ...
func CreateServer(serverKeyFile, serverCertFile, clientRootCA string) *tls.Config {
	if c, e := CreateServerE(serverKeyFile, serverCertFile, clientRootCA); e != nil {
		panic("failed to create CreateServer " + e.Error())
	} else {
		return c
	}
}

// CreateServerE ...
func CreateServerE(serverKeyFile, serverCertFile, clientRootCA string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, err
	}

	c := &tls.Config{Certificates: []tls.Certificate{cert}}

	if clientRootCA != "" {
		rootCA, err := TLSLoadPermFile(clientRootCA)
		if err != nil {
			return nil, fmt.Errorf("failed to read clientCertFile %s, error %v", clientRootCA, err)
		}

		c.ClientAuth = tls.RequireAndVerifyClientCert
		c.ClientCAs = x509.NewCertPool()
		c.ClientCAs.AddCert(rootCA)
	}

	return c, nil
}

// CreateClient ...
// if serverRootCA is empty, the client will will not check the root CA of client.
func CreateClient(clientKeyFile, clientCertFile, serverRootCA string) *tls.Config {
	if c, e := CreateClientE(clientKeyFile, clientCertFile, serverRootCA); e != nil {
		panic("failed to create CreateClient " + e.Error())
	} else {
		return c
	}
}

// CreateClientE ...
func CreateClientE(clientKeyFile, clientCertFile, serverRootCA string) (*tls.Config, error) {
	c := &tls.Config{}
	if serverRootCA == "" {
		c.InsecureSkipVerify = true // #nosec G402
	} else {
		rootCA, err := TLSLoadPermFile(serverRootCA)
		if err != nil {
			return nil, fmt.Errorf("failed to read clientCertFile %s, error %v", serverRootCA, err)
		}

		c.RootCAs = x509.NewCertPool()
		c.RootCAs.AddCert(rootCA)

		SkipHostnameVerification(c)
	}

	if clientKeyFile != "" && clientCertFile != "" {
		cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
		if err != nil {
			return nil, err
		}

		c.Certificates = []tls.Certificate{cert}
	}

	return c, nil
}

// CreateClientBytes ...
func CreateClientBytes(clientKeyFile, clientCertFile, serverRootCA []byte) *tls.Config {
	if c, e := CreateClientBytesE(clientKeyFile, clientCertFile, serverRootCA); e != nil {
		panic("failed to create CreateClient " + e.Error())
	} else {
		return c
	}
}

// CreateClientBytesE ...
func CreateClientBytesE(clientKeyFile, clientCertFile, serverRootCA []byte) (*tls.Config, error) {
	c := &tls.Config{}
	if len(serverRootCA) == 0 {
		c.InsecureSkipVerify = true // #nosec G402
	} else {
		rootCA, err := TLSLoadPerm(serverRootCA)
		if err != nil {
			return nil, fmt.Errorf("failed to read clientCertBytes %s, error %v", string(serverRootCA), err)
		}

		c.RootCAs = x509.NewCertPool()
		c.RootCAs.AddCert(rootCA)

		SkipHostnameVerification(c)
	}

	if len(clientKeyFile) > 0 && len(clientCertFile) > 0 {
		cert, err := tls.X509KeyPair(clientCertFile, clientKeyFile)
		if err != nil {
			return nil, err
		}

		c.Certificates = []tls.Certificate{cert}
	}

	return c, nil
}

// CreateServerBytes ....
func CreateServerBytes(serverKeyFile, serverCertFile, clientRootCA []byte) *tls.Config {
	if c, e := CreateServerBytesE(serverKeyFile, serverCertFile, clientRootCA); e != nil {
		panic("failed to create CreateServer " + e.Error())
	} else {
		return c
	}
}

// CreateServerBytesE ...
func CreateServerBytesE(serverKeyFile, serverCertFile, clientRootCA []byte) (*tls.Config, error) {
	cert, err := tls.X509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, err
	}

	c := &tls.Config{Certificates: []tls.Certificate{cert}}

	if len(clientRootCA) > 0 {
		rootCA, err := TLSLoadPerm(clientRootCA)
		if err != nil {
			return nil, fmt.Errorf("failed to read clientCert %s, error %v", string(clientRootCA), err)
		}

		c.ClientAuth = tls.RequireAndVerifyClientCert
		c.ClientCAs = x509.NewCertPool()
		c.ClientCAs.AddCert(rootCA)
	}

	return c, nil
}

// SkipHostnameVerification ...
// nolint
// https://github.com/digitalbitbox/bitbox-wallet-app/blob/b04bd07852d5b37939da75b3555b5a1e34a976ee/backend/coins/btc/electrum/electrum.go#L76-L111
func SkipHostnameVerification(c *tls.Config) {
	c.InsecureSkipVerify = true // #nosec G402
	// Not actually skipping, we check the cert in VerifyPeerCertificate
	c.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
		// Code copy/pasted and adapted from
		// nolint
		// https://github.com/golang/go/blob/81555cb4f3521b53f9de4ce15f64b77cc9df61b9/src/crypto/tls/handshake_client.go#L327-L344, but adapted to skip the hostname verification.
		// See https://github.com/golang/go/issues/21971#issuecomment-412836078.

		// If this is the first handshake on a connection, process and
		// (optionally) verify the server's certificates.
		certs := make([]*x509.Certificate, len(rawCerts))
		for i, asn1Data := range rawCerts {
			cert, err := x509.ParseCertificate(asn1Data)
			if err != nil {
				return fmt.Errorf("bitbox/electrum: failed to parse certificate from server: %v ", err.Error())
			}
			certs[i] = cert
		}

		opts := x509.VerifyOptions{
			Roots:         c.RootCAs,
			CurrentTime:   time.Now(),
			DNSName:       "", // <- skip hostname verification
			Intermediates: x509.NewCertPool(),
		}

		for i, cert := range certs {
			if i == 0 {
				continue
			}
			opts.Intermediates.AddCert(cert)
		}
		_, err := certs[0].Verify(opts)
		return err
	}
}

// TLSLoadPermFile ...
func TLSLoadPermFile(rootCAFile string) (*x509.Certificate, error) {
	caStr, err := ioutil.ReadFile(rootCAFile)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(caStr)

	if block == nil {
		return nil, err
	}

	if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
		return nil, fmt.Errorf("decode ca block file fail")
	}

	return x509.ParseCertificate(block.Bytes)
}

// TLSLoadPerm ...
func TLSLoadPerm(rootCAFile []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(rootCAFile)
	if block == nil {
		return nil, errors.New("decode ca file error")
	}

	if block.Type != "CERTIFICATE" || len(block.Headers) != 0 {
		return nil, fmt.Errorf("decode ca block file fail")
	}

	return x509.ParseCertificate(block.Bytes)
}
