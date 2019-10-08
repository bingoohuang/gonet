package gonet_test

import (
	"testing"

	"github.com/bingoohuang/gonet"
	"github.com/stretchr/testify/assert"
)

type GetRsp struct {
	Origin string `json:"origin"`
	URL    string `json:"url"`
}

func TestRestGet(t *testing.T) {
	var rsp GetRsp
	url := `https://httpbin.org/get`
	err := gonet.RestGet(url, &rsp)
	assert.Nil(t, err)
	assert.Equal(t, url, rsp.URL)
}
