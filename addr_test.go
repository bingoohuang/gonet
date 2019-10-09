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

	local, error = IsLocalAddr("unkownhost")
	assert.NotNil(t, error)
	assert.False(t, local)
}
