package gonet

import (
	"fmt"
	"net"
	"strconv"
)

// IsPortFree tells whether the port is free or not
func IsPortFree(port int) bool {
	l, err := ListenPort(port)
	if err != nil {
		fmt.Println(err)
		return false
	}

	_ = l.Close()

	return true
}

// FreePort asks the kernel for a free open port that is ready to use.
func FreePort() (int, error) {
	l, e := ListenPort(0)
	if e != nil {
		return 0, e
	}

	_ = l.Close()

	return l.Addr().(*net.TCPAddr).Port, nil
}

// MustFreePort is deprecated, use FreePort instead
// Ask the kernel for a free open port that is ready to use
func MustFreePort() int {
	port, err := FreePort()
	if err != nil {
		panic(err)
	}

	return port
}

// MustFreePortStr get a free port as string.
func MustFreePortStr() string {
	return strconv.Itoa(MustFreePort())
}

// FreePorts asks the kernel for free open ports that are ready to use.
func FreePorts(count int) ([]int, error) {
	ports := make([]int, count)

	for i := 0; i < count; i++ {
		l, err := ListenPort(0)
		if err != nil {
			return nil, err
		}
		defer l.Close()
		ports[i] = l.Addr().(*net.TCPAddr).Port
	}

	return ports, nil
}

// ListenPort listens on port
func ListenPort(port int) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf(":%d", port))
}

// FindFreePortFrom finds a free port from starting port
func FindFreePortFrom(starting int) int {
	p := starting
	for ; !IsPortFree(p); p++ {
	}

	return p
}
