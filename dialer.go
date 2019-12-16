package gonet

import (
	"context"
	"net"
	"net/http"
	"time"
)

// DialFn was defined to make code more readable.
type DialFn func(network, addr string) (c net.Conn, err error)

// DialContextFn was defined to make code more readable.
type DialContextFn func(ctx context.Context, network, address string) (net.Conn, error)

// DialerTimeout  implements our own dialer in order to set read and write idle timeouts.
func DialerTimeout(rwtimeout, ctimeout time.Duration) DialFn {
	dialer := &net.Dialer{Timeout: ctimeout}

	return func(network, addr string) (net.Conn, error) {
		c, err := dialer.Dial(network, addr)
		if err != nil {
			return nil, err
		}

		if rwtimeout > 0 {
			return &tcpConn{TCPConn: c.(*net.TCPConn), timeout: rwtimeout}, nil
		}

		return c, nil
	}
}

// DialContextTimeout implements our own dialer in order to set read and write idle timeouts.
func DialContextTimeout(rwtimeout, ctimeout time.Duration) DialContextFn {
	dialer := &net.Dialer{Timeout: ctimeout}

	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		c, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			return nil, err
		}

		if rwtimeout > 0 {
			return &tcpConn{TCPConn: c.(*net.TCPConn), timeout: rwtimeout}, nil
		}

		return c, nil
	}
}

// tcpConn is our own net.Conn which sets a read and write deadline and resets them each
// time there is read or write activity in the connection.
type tcpConn struct {
	*net.TCPConn
	timeout time.Duration
}

func (c *tcpConn) Read(b []byte) (int, error) {
	if err := c.TCPConn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}

	return c.TCPConn.Read(b)
}

func (c *tcpConn) Write(b []byte) (int, error) {
	if err := c.TCPConn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return 0, err
	}

	return c.TCPConn.Write(b)
}

// DefaultClient returns a default client with sensible values for slow 3G connections and above.
func DefaultClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialContext:           DialContextTimeout(30*time.Second, 10*time.Second),
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          100,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}
}
