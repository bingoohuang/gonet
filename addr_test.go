package gonet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLocal(t *testing.T) {
	local, error := IsLocalAddr("127.0.0.1")
	assert.Nil(t, error)
	assert.True(t, local)

	local, error = IsLocalAddr("localhost")
	assert.Nil(t, error)
	assert.True(t, local)

	//local, error = IsLocalAddr("192.168.162.108")
	//assert.Nil(t, error)
	//assert.True(t, local)

	//local, error = IsLocalAddr("fe80::c0b:c8d7:5739:2605")
	//assert.Nil(t, error)
	//assert.True(t, local)

	local, error = IsLocalAddr("unknown.host")
	assert.NotNil(t, error)
	assert.False(t, local)
}
