package gonet

import (
	"bufio"
	"crypto/tls"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateTLSListenerV1(t *testing.T) {
	create("", "", t)
}
func TestCreateTLSListenerV2(t *testing.T) {
	create("tls_test_files/client.key", "tls_test_files/client.pem", t)
}

func create(clientKey, clientCrt string, t *testing.T) {
	tlsConfig := MustCreateServerTLSConfig("tls_test_files/server.key", "tls_test_files/server.pem", clientCrt)
	portStr := MustFreePortStr()
	ln, err := tls.Listen("tcp", ":"+portStr, tlsConfig)
	if err != nil {
		assert.Error(t, err)
		return
	}

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			assert.Error(t, err)
			return
		}
		handleConn(conn)
		conn.Close()
	}()

	client(clientKey, clientCrt, portStr, t)
}

func client(clientKey, clientCrt string, portStr string, t *testing.T) {
	conn, err := tls.Dial("tcp", "127.0.0.1:"+portStr, MustCreateClientTLSConfig(clientKey, clientCrt))
	if err != nil {
		assert.Error(t, err)
		return
	}
	defer conn.Close()
	_, err = conn.Write([]byte("hello\n"))
	if err != nil {
		assert.Error(t, err)
		return
	}
	buf := make([]byte, 100)
	n, err := conn.Read(buf)
	if err != nil {
		assert.Error(t, err)
		return
	}
	println(buf[:n])
}

func handleConn(conn net.Conn) {
	defer conn.Close()
	r := bufio.NewReader(conn)
	for {
		msg, err := r.ReadString('\n')
		if err != nil {
			log.Println(err)
			return
		}
		println(msg)
		n, err := conn.Write([]byte("world\n"))
		if err != nil {
			log.Println(n, err)
			return
		}
	}
}
