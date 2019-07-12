package gonet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIP(t *testing.T) {
	a := assert.New(t)
	a.True(IsIP("192.168.0.1"), "192.168.0.1是IPv4地址")
	a.True(IsIP("FE80::0202:B3FF:FE1E:8329"), "FE80::0202:B3FF:FE1E:8329是IPv6地址")
	a.True(IsIP("2001:db8::68"), "2001:db8::68是IPv6地址")
	a.False(IsIP("http://[2001:db8:0:1]:80"), "http://[2001:db8:0:1]:80不是IP地址")
	a.False(IsIP("app01"), "app01不是IP地址")
	a.False(IsIP("app.01"), "app.01不是IP地址")
}
