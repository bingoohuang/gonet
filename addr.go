package gonet

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// IsLocalAddr 判断addr（ip，域名等）是否指向本机
// 由于IP可能经由iptable指向，或者可能是域名，或者其它，不能直接与本机IP做对比
// 本方法构建一个临时的HTTP服务，然后使用指定的addr去连接改HTTP服务，如果能连接上，说明addr是指向本机的地址
func IsLocalAddr(addr string) (bool, error) {
	if addr == "localhost" || addr == "127.0.0.1" || addr == "::1" {
		return true, nil
	}

	if _, ok := ListIPMap()[addr]; ok {
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

	go func() { _ = server.ListenAndServe() }()

	time.Sleep(100 * time.Millisecond) // nolint gomnd

	resp, err := HTTPGet(`http://` + JoinHostPort(addr, port))

	_ = server.Close()

	if err != nil {
		return false, err
	}

	return string(resp) == radStr, nil
}

// JoinHostPort make IP:Port for ipv4/domain or [IPv6]:Port for ipv6.
func JoinHostPort(host string, port int) string {
	if IsIPv6(host) {
		return fmt.Sprintf("[%s]:%d", host, port)
	}

	return fmt.Sprintf("%s:%d", host, port)
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
