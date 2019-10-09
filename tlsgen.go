package gonet

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func TLSGenRootFiles(path, outKey, outPem string) error {
	_, rootKey, rootDerBytes, e := TLSGenRootPem()
	if e != nil {
		return e
	}

	if err := keyToFile(filepath.Join(path, outKey), rootKey); err != nil {
		return err
	}

	return certToFile(filepath.Join(path, outPem), rootDerBytes)
}

func TLSGenServerFiles(path, rootKey, rootPem, host, outKey, outPem string) error {
	rootPrivate, ca, e := TLSLoadKeyPair(path, rootPem, rootKey)
	if e != nil {
		return e
	}

	leafKey, derBytes, err := TLSGenServerPem(host, ca, rootPrivate)
	if err != nil {
		return err
	}

	if err := keyToFile(filepath.Join(path, outKey), leafKey); err != nil {
		return err
	}

	return certToFile(filepath.Join(path, outPem), derBytes)
}

func TLSGenClientFiles(path, rootKey, rootPem, outKey, outPem string) error {
	rootPrivate, ca, e := TLSLoadKeyPair(path, rootPem, rootKey)
	if e != nil {
		return e
	}

	clientKey, derBytes, e := TLSGenClientPem(ca, rootPrivate)
	if e != nil {
		return e
	}

	if err := keyToFile(filepath.Join(path, outKey), clientKey); err != nil {
		return err
	}

	return certToFile(filepath.Join(path, outPem), derBytes)
}

func TLSGenAll(path, host string) error {
	ca, rootKey, rootPrivate, err := TLSGenRootPem()
	if err != nil {
		return err
	}

	if err := keyToFile(filepath.Join(path, "root.key"), rootKey); err != nil {
		return err
	}
	if err := certToFile(filepath.Join(path, "root.pem"), rootPrivate); err != nil {
		return err
	}

	leafKey, leafPem, err := TLSGenServerPem(host, ca, rootKey)
	if err != nil {
		return err
	}

	if err := keyToFile(filepath.Join(path, "server.key"), leafKey); err != nil {
		return err
	}
	if err := certToFile(filepath.Join(path, "server.pem"), leafPem); err != nil {
		return err
	}
	clientKey, clientPem, e := TLSGenClientPem(ca, rootKey)
	if e != nil {
		return e
	}

	if err := keyToFile(filepath.Join(path, "client.key"), clientKey); err != nil {
		return err
	}
	if err := certToFile(filepath.Join(path, "client.pem"), clientPem); err != nil {
		return err
	}
	/*
	   `Successfully generated certificates! Here's what you generated.
	   # Root CA
	   root.key
	   	The private key for the root Certificate Authority. Keep this private.
	   root.pem
	   	The public key for the root Certificate Authority. Clients should load the
	   	certificate in this file to connect to the server.
	   root.debug.crt
	   	Debug information about the generated certificate.
	   # Server Certificate - Use these to serve TLS traffic.
	   server.key
	   	Private key (PEM-encoded) for terminating TLS traffic on the server.
	   server.pem
	   	Public key for terminating TLS traffic on the server.
	   server.debug.crt
	   	Debug information about the generated certificate
	   # Client Certificate - You probably don't need these.
	   client.key: Secret key for TLS client authentication
	   client.pem: Public key for TLS client authentication
	   See https://github.com/Shyp/generate-tls-cert for examples of how to use in code.
	   `)
	*/
	return nil
}

func TLSLoadKeyPair(path string, rootPem string, rootKey string) (crypto.PrivateKey, *x509.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(filepath.Join(path, rootPem), filepath.Join(path, rootKey))
	if err != nil {
		return nil, nil, err
	}
	rootPrivateKey := cert.PrivateKey
	ca, err := x509.ParseCertificate(cert.Certificate[0])
	return rootPrivateKey, ca, err
}

func TLSGenRootPem() (*x509.Certificate, *ecdsa.PrivateKey, []byte, error) {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit) //nolint G404
	if err != nil {
		return nil, nil, nil, err
	}
	rootKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, nil, err
	}

	rootTemplate := &x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"BJCA"}, CommonName: "Root CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Date(2049, 12, 31, 23, 59, 59, 0, time.UTC),
		KeyUsage:              x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	derBytes, err := x509.CreateCertificate(rand.Reader, rootTemplate, rootTemplate, &rootKey.PublicKey, rootKey)
	return rootTemplate, rootKey, derBytes, err
}

func TLSGenServerPem(host string, ca *x509.Certificate, key crypto.PrivateKey) (*ecdsa.PrivateKey, []byte, error) {
	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	var serialNumber *big.Int
	serialNumber, err = rand.Int(rand.Reader, serialNumberLimit)//nolint G404
	if err != nil {
		return nil, nil, err
	}

	leafTemplate := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"BJCA"}, CommonName: "leaf"},
		NotBefore:             time.Now(),
		NotAfter:              time.Date(2049, 12, 31, 23, 59, 59, 0, time.UTC),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	hosts := strings.Split(host, ",")
	for _, h := range hosts {
		if h == "" {
			continue
		}
		if ip := net.ParseIP(h); ip != nil {
			leafTemplate.IPAddresses = append(leafTemplate.IPAddresses, ip)
		} else {
			leafTemplate.DNSNames = append(leafTemplate.DNSNames, h)
		}
	}
	var derBytes []byte
	derBytes, err = x509.CreateCertificate(rand.Reader, &leafTemplate, ca, &leafKey.PublicKey, key)
	return leafKey, derBytes, err
}

func TLSGenClientPem(rootCA *x509.Certificate, rootPrivate crypto.PrivateKey) (*ecdsa.PrivateKey, []byte, error) {
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	clientTemplate := x509.Certificate{
		SerialNumber:          new(big.Int).SetInt64(4),
		Subject:               pkix.Name{Organization: []string{"BJCA"}, CommonName: "client"},
		NotBefore:             time.Now(),
		NotAfter:              time.Date(2049, 12, 31, 23, 59, 59, 0, time.UTC),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}
	var derBytes []byte
	derBytes, err = x509.CreateCertificate(rand.Reader, &clientTemplate, rootCA, &clientKey.PublicKey, rootPrivate)
	return clientKey, derBytes, err
}

// keyToFile writes a PEM serialization of |key| to a new file called |filename|.
func keyToFile(filename string, key *ecdsa.PrivateKey) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	b, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		//fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
		return err
	}

	return pem.Encode(file, &pem.Block{Type: "EC PRIVATE KEY", Bytes: b})
}

func certToFile(filename string, derBytes []byte) error {
	certOut, err := os.Create(filename)
	if err != nil {
		//log.Fatalf("failed to open cert.pem for writing: %s", err)
		return err
	}
	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		// log.Fatalf("failed to write data to cert.pem: %s", err)
		return err
	}
	return certOut.Close()
}
