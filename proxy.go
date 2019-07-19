package gonet

import (
	"net/http"
	"net/http/httputil"
	"strings"
	"time"
)

func ReverseProxy(originalPath, targetHost, targetPath string) *httputil.ReverseProxy {
	director := func(req *http.Request) {
		req.URL.Scheme = "http"

		req.URL.Host = targetHost
		req.URL.Path = targetPath

		req.Header.Add("X-Forwarded-Host", req.Host)
		req.Header.Add("X-Origin-Host", req.Header.Get("Host"))
	}

	modifyResponse := func(r *http.Response) error {
		if r.StatusCode == http.StatusMovedPermanently || r.StatusCode == http.StatusFound {
			// 301 302时，改写Location返回头
			basePath := strings.TrimRight(originalPath, targetPath)
			r.Header.Set("Location", basePath+r.Header.Get("Location"))
		}

		return nil
	}
	transport := &http.Transport{DialContext: TimeoutDialer(15*time.Second, 15*time.Second)}

	// 更多可以参见 https://github.com/Integralist/go-reverse-proxy/blob/master/proxy/proxy.go
	return &httputil.ReverseProxy{Director: director, ModifyResponse: modifyResponse, Transport: transport}
}
