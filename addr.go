package gonet

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// IsLocalAddr 判断addr（ip，域名等）是否指向本机
// 由于IP可能经由iptable指向，或者可能是域名，或者其它，不能直接与本机IP做对比
// 本方法构建一个临时的HTTP服务，然后使用指定的addr去连接改HTTP服务，如果能连接上，说明addr是指向本机的地址
func IsLocalAddr(addr string) (bool, error) {
	if addr == "localhost" || addr == "127.0.0.1" || addr == "::1" {
		return true, nil
	}

	localIPMap := ListIPMap()
	if _, ok := localIPMap[addr]; ok {
		return true, nil
	}

	port, err := FreePort()
	if err != nil {
		return false, err
	}

	radStr := RandString(512) // nolint gomnd
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, radStr)
	})

	server := &http.Server{Addr: fmt.Sprintf(":%d", port), Handler: mux}
	exitChan := make(chan bool)

	go func() {
		err := server.ListenAndServe()
		logrus.Infof("ListenAndServe %v", err)
		exitChan <- true
	}()

	url := fmt.Sprintf("http://%s:%d", addr, port)
	resp, err := HTTPGet(url)

	if e := server.Close(); e != nil {
		logrus.Warnf("Close %v", err)
	}

	if err != nil {
		logrus.Warnf("HTTPGet %v", err)
		return false, err
	}

	select {
	case <-time.After(10 * time.Second): // nolint gomnd
	case <-exitChan:
	}

	return string(resp) == radStr, nil
}

// https://stackoverflow.com/a/31832326
const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

// RandString ...
func RandString(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; { // nolint gomnd
		if remain == 0 {
			cache, remain = rand.Int63(), letterIdxMax
		}

		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}

		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}
