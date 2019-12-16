package gonet

import (
	"context"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/proxy"
)

// DialerTimeoutBean ...
type DialerTimeoutBean struct {
	ReadWriteTimeout time.Duration
	ConnTimeout      time.Duration
}

var _ proxy.Dialer = (*DialerTimeoutBean)(nil)

// DialContext ...
func (d DialerTimeoutBean) DialContext(ctx context.Context, network, addr string) (c net.Conn, err error) {
	dialer := &net.Dialer{Timeout: d.ConnTimeout}
	c, err = dialer.DialContext(ctx, network, addr)

	if err != nil {
		return nil, err
	}

	if d.ReadWriteTimeout > 0 {
		c = &tcpConn{TCPConn: c.(*net.TCPConn), timeout: d.ReadWriteTimeout}
	}

	return c, nil
}

// Dial ...
func (d DialerTimeoutBean) Dial(network, addr string) (c net.Conn, err error) {
	dialer := &net.Dialer{Timeout: d.ConnTimeout}
	c, err = dialer.Dial(network, addr)

	if err != nil {
		return nil, err
	}

	if d.ReadWriteTimeout > 0 {
		c = &tcpConn{TCPConn: c.(*net.TCPConn), timeout: d.ReadWriteTimeout}
	}

	return c, nil
}

// DialContextFn was defined to make code more readable.
type DialContextFn func(ctx context.Context, network, address string) (net.Conn, error)

// DialerTimeout  implements our own dialer in order to set read and write idle timeouts.
func DialerTimeout(rwtimeout, ctimeout time.Duration) func(network, addr string) (c net.Conn, err error) {
	return DialerTimeoutBean{ReadWriteTimeout: rwtimeout, ConnTimeout: ctimeout}.Dial
}

// DialContextTimeout implements our own dialer in order to set read and write idle timeouts.
func DialContextTimeout(rwtimeout, ctimeout time.Duration) func(ctx context.Context, network, addr string) (net.Conn, error) { // nolint
	return DialerTimeoutBean{ReadWriteTimeout: rwtimeout, ConnTimeout: ctimeout}.DialContext
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
