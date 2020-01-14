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

//func TestHTTPGet(t *testing.T) {
//	url := "http://127.0.0.1:9901"
//	for i := 0; i < 10000; i++ {
//		r, _ := gonet.HTTPGet(url)
//		fmt.Println(string(r))
//		time.Sleep(500 * time.Millisecond)
//	}
//}
