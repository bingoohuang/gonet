package gonet

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

func IsLocalAddr(addr string) (bool, error) {
	port, err := FreePort()
	if err != nil {
		return false, err
	}

	radStr := RandString(512)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprintf(w, radStr)
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
	case <-time.After(10 * time.Second):
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

func RandString(n int) string {
	b := make([]byte, n)
	// A rand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, rand.Int63(), letterIdxMax; i >= 0; {
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
