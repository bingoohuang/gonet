package gonet

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"time"
)

func TLSConfigCreateServerMust(serverKeyFile, serverCertFile, clientRootCA string) *tls.Config {
	if c, e := TLSConfigCreateServer(serverKeyFile, serverCertFile, clientRootCA); e != nil {
		panic("failed to create TLSConfigCreateServer " + e.Error())
	} else {
		return c
	}
}

func TLSConfigCreateServer(serverKeyFile, serverCertFile, clientRootCA string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(serverCertFile, serverKeyFile)
	if err != nil {
		return nil, err
	}

	c := &tls.Config{Certificates: []tls.Certificate{cert}}
	if clientRootCA == "" {
		return c, nil
	}

	rootCA, err := TLSLoadPermFile(clientRootCA)
	if err != nil {
		return nil, fmt.Errorf("failed to read clientCertFile %s, error %v", clientRootCA, err)
	}

	c.ClientAuth = tls.RequireAndVerifyClientCert
	c.ClientCAs = x509.NewCertPool()
	c.ClientCAs.AddCert(rootCA)

	return c, nil
}

func TLSConfigCreateClientMust(clientKeyFile, clientCertFile, clientRootCA string) *tls.Config {
	if c, e := TLSConfigCreateClient(clientKeyFile, clientCertFile, clientRootCA); e != nil {
		panic("failed to create TLSConfigCreateClient " + e.Error())
	} else {
		return c
	}
}

func TLSConfigCreateClient(clientKeyFile, clientCertFile, clientRootCA string) (*tls.Config, error) {
	c := &tls.Config{}
	if clientKeyFile == "" || clientCertFile == "" || clientRootCA == "" {
		c.InsecureSkipVerify = true // #nosec G402
		return c, nil
	}

	rootCA, err := TLSLoadPermFile(clientRootCA)
	if err != nil {
		return nil, fmt.Errorf("failed to read clientCertFile %s, error %v", clientRootCA, err)
	}

	c.RootCAs = x509.NewCertPool()
	c.RootCAs.AddCert(rootCA)

	SkipHostnameVerification(c)

	cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		return nil, err
	}

	c.Certificates = []tls.Certificate{cert}
	return c, nil
}

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
