package tlsconf

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/bingoohuang/gg/pkg/netx/freeport"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTLSGenRootFiles(t *testing.T) {
	assert.Nil(t, os.MkdirAll("v0", 0777))
	assert.Nil(t, TLSGenRootFiles("v0", "root.key", "root.pem"))
}

func TestTlsCertsGenv1(t *testing.T) {
	assert.Nil(t, os.MkdirAll("v1", 0777))
	defer os.RemoveAll("v1")

	assert.Nil(t, TLSGenRootFiles("v1", "root.key", "root.pem"))
	assert.Nil(t, TLSGenServerFiles("v1", "root.key", "root.pem", "",
		"server.key", "server.pem"))
	assert.Nil(t, TLSGenClientFiles("v1", "root.key", "root.pem",
		"client.key", "client.pem"))

	// 提供https，但是客户端/服务端都不校验彼此证书
	create("", "", "v1/server.key", "v1/server.pem", "", "", t)
	// 客户端校验服务端证书, 服务端不校验客户端证书
	create("v1/root.pem", "", "v1/server.key", "v1/server.pem", "", "", t)
	// 客户端不校验服务端证书，服务端校验客户端证书
	create("", "v1/root.pem", "v1/server.key", "v1/server.pem", "v1/client.key", "v1/client.pem", t)
}

func TestTlsCertsGenv2(t *testing.T) {
	assert.Nil(t, os.MkdirAll("v2", 0777))
	defer os.RemoveAll("v2")

	assert.Nil(t, TLSGenAll("v2", ""))

	// 相互校验
	create("v2/root.pem", "v2/root.pem",
		"v2/server.key", "v2/server.pem",
		"v2/client.key", "v2/client.pem", t)
}

func execCertTest() {
	cmd := exec.Command("/bin/bash", "-c", "testdata/cert_test.sh")
	out, err := cmd.CombinedOutput()

	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}

	fmt.Printf("combined out:\n%s\n", string(out))
}

func TestTlsCertsGenv3(t *testing.T) {
	assert.Nil(t, os.MkdirAll("v3", 0777))
	defer os.RemoveAll("v3")
	execCertTest()

	create("v3/root.pem", "v3/root.pem",
		"v3/server.key", "v3/server.pem",
		"v3/client.key", "v3/client.pem", t)
}

func create(serverRootCA, clientRoot, serverKey, serverCrt, clientKey, clientCrt string, t *testing.T) {
	tlsConfig := CreateServer(serverKey, serverCrt, clientRoot)
	port := freeport.Port()
	ln, err := tls.Listen("tcp", fmt.Sprintf(":%d", port), tlsConfig)

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

		handleConn(conn, t)
	}()

	client(serverRootCA, clientKey, clientCrt, port, t)
}

func client(serverRootCA, clientKey, clientCrt string, port int, t *testing.T) {
	add := fmt.Sprintf("127.0.0.1:%d", port)
	tlsConfig := CreateClient(clientKey, clientCrt, serverRootCA)
	conn, err := tls.Dial("tcp", add, tlsConfig)
	assert.Nil(t, err)

	defer conn.Close()
	_, err = conn.Write([]byte("hello\n"))
	assert.Nil(t, err)

	buf := make([]byte, 100)
	n, err := conn.Read(buf)
	assert.True(t, err == nil || err == io.EOF)

	println(string(buf[:n]))
}

func handleConn(conn net.Conn, t *testing.T) {
	defer conn.Close()
	r := bufio.NewReader(conn)

	msg, err := r.ReadString('\n')
	assert.Nil(t, err)
	println(msg)

	_, err = conn.Write([]byte("world\n"))
	assert.Nil(t, err)
}
