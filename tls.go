package gonet

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

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
