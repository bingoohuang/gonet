package freeport

import (
	"net"
	"strconv"
)

// FreePort asks the kernel for a free open port that is ready to use.
func FreePort() (int, error) {
	l, e := listenPort0()
	if e != nil {
		return 0, e
	}
	port := l.Addr().(*net.TCPAddr).Port
	_ = l.Close()
	return port, nil
}

// Port is deprecated, use FreePort instead
// Ask the kernel for a free open port that is ready to use
func Port() int {
	port, err := FreePort()
	if err != nil {
		panic(err)
	}
	return port
}

// PortString get a free port as string.
func PortString() string {
	return strconv.Itoa(Port())
}

// FreePort asks the kernel for free open ports that are ready to use.
func FreePorts(count int) ([]int, error) {
	ports := make([]int, count)

	for i := 0; i < count; i++ {
		l, err := listenPort0()
		if err != nil {
			return nil, err
		}
		defer l.Close()
		ports[i] = l.Addr().(*net.TCPAddr).Port
	}

	return ports, nil
}

func listenPort0() (*net.TCPListener, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return nil, err
	}
	return net.ListenTCP("tcp", addr)
}
