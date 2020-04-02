package freeport

import (
	"github.com/stretchr/testify/assert"

	"net"
	"strconv"
	"testing"
)

func TestMustFreePort(t *testing.T) {
	port := Port()
	assert.True(t, IsPortFree(port))
}

func BenchmarkFreePort(b *testing.B) {
	for i := 0; i < b.N; i++ {
		port := Port()
		assert.True(b, port > 0)
	}
}

func TestMustFreePortStr(t *testing.T) {
	assert.NotEmpty(t, PortStr())
}

func TestGetFreePort(t *testing.T) {
	port, err := PortE()
	if err != nil {
		t.Error(err)
	}

	if port == 0 {
		t.Error("port == 0")
	}

	// Try to listen on the port
	l, err := net.Listen("tcp", "localhost"+":"+strconv.Itoa(port))
	if err != nil {
		t.Error(err)
	}

	defer l.Close()
}

func TestGetFreePorts(t *testing.T) {
	count := 3
	ports, err := Ports(count)

	if err != nil {
		t.Error(err)
	}

	if len(ports) == 0 {
		t.Error("len(ports) == 0")
	}

	for _, port := range ports {
		if port == 0 {
			t.Error("port == 0")
		}

		// Try to listen on the port
		l, err := net.Listen("tcp", "localhost"+":"+strconv.Itoa(port))
		if err != nil {
			t.Error(err)
		}
		defer l.Close()
	}
}

func TestFindFreePortFrom(t *testing.T) {
	p := FindPortFrom(1024)   // nolint gomnd
	assert.True(t, p >= 1024) // nolint gomnd
}
