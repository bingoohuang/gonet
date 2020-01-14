package gonet

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"
)

// ReqOption ...
type ReqOption struct {
	ShowDebug        bool
	EnableCookie     bool
	Gzip             bool
	DumpBody         bool
	UserAgent        string
	ConnectTimeout   time.Duration
	ReadWriteTimeout time.Duration
	TLSClientConfig  *tls.Config
	CookieJar        *cookiejar.Jar
	Proxy            func(*http.Request) (*url.URL, error)
	Transport        http.RoundTripper
}

// NewCookieJar creates a cookiejar to store cookies.
func NewCookieJar() *cookiejar.Jar {
	cookieJar, _ := cookiejar.New(nil)
	return cookieJar
}

// NewReqOption creates a default settings
func NewReqOption() *ReqOption {
	return &ReqOption{
		ShowDebug:        false,
		UserAgent:        "Gonet",
		ConnectTimeout:   10 * time.Second, // nolint gomnd
		ReadWriteTimeout: 10 * time.Second, // nolint gomnd
		TLSClientConfig:  nil,
		Proxy:            nil,
		Transport:        nil,
		EnableCookie:     false,
		Gzip:             true,
		DumpBody:         true,
	}
}

// Req return *HTTPReq with specific method
func (s *ReqOption) Req(rawURL, method string) (*HTTPReq, error) {
	var resp http.Response

	u, err := url.Parse(rawURL)

	if err != nil {
		return nil, err
	}

	req := http.Request{
		URL:        u,
		Method:     method,
		Header:     make(http.Header),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1, // nolint gomnd
		ProtoMinor: 1, // nolint gomnd
	}

	return &HTTPReq{
		url:     rawURL,
		req:     &req,
		params:  map[string]string{},
		files:   map[string]string{},
		setting: *s,
		resp:    &resp,
		body:    nil,
	}, nil
}

// HTTPReq provides more useful methods for requesting one url than http.Request.
type HTTPReq struct {
	url     string
	req     *http.Request
	params  map[string]string
	files   map[string]string
	setting ReqOption
	resp    *http.Response
	body    []byte
	dump    []byte
}

// BasicAuth sets the request's Authorization header
// to use HTTP Basic Authentication with the provided username and password.
func (b *HTTPReq) BasicAuth(username, password string) *HTTPReq {
	b.req.SetBasicAuth(username, password)
	return b
}

// CookieJar sets enable/disable cookiejar
func (b *HTTPReq) CookieJar(jar *cookiejar.Jar) *HTTPReq {
	b.setting.EnableCookie = true
	b.setting.CookieJar = jar

	return b
}

// EnableCookie sets enable/disable cookiejar
func (b *HTTPReq) EnableCookie(enable bool) *HTTPReq {
	b.setting.EnableCookie = enable

	return b
}

// UserAgent sets User-Agent header field
func (b *HTTPReq) UserAgent(useragent string) *HTTPReq {
	b.setting.UserAgent = useragent

	return b
}

// Debug sets show debug or not when executing request.
func (b *HTTPReq) Debug(isdebug bool) *HTTPReq {
	b.setting.ShowDebug = isdebug

	return b
}

// DumpBody ...
func (b *HTTPReq) DumpBody(isdump bool) *HTTPReq {
	b.setting.DumpBody = isdump

	return b
}

// DumpRequest returns the DumpRequest
func (b *HTTPReq) DumpRequest() []byte {
	return b.dump
}

// DumpRequestString returns the DumpRequest string
func (b *HTTPReq) DumpRequestString() string {
	return string(b.DumpRequest())
}

// Timeout sets connect time out and read-write time out for Request.
func (b *HTTPReq) Timeout(connectTimeout, readWriteTimeout time.Duration) *HTTPReq {
	b.setting.ConnectTimeout = connectTimeout
	b.setting.ReadWriteTimeout = readWriteTimeout

	return b
}

// TLSClientConfig sets tls connection configurations if visiting https url.
func (b *HTTPReq) TLSClientConfig(config *tls.Config) *HTTPReq {
	b.setting.TLSClientConfig = config

	return b
}

// Header add header item string in request.
func (b *HTTPReq) Header(key, value string) *HTTPReq {
	b.req.Header.Set(key, value)

	return b
}

// Host Set HOST
func (b *HTTPReq) Host(host string) *HTTPReq {
	b.req.Host = host

	return b
}

// ProtocolVersion set the protocol version for incoming requests.
// Client requests always use HTTP/1.1.
func (b *HTTPReq) ProtocolVersion(vers string) *HTTPReq {
	if vers == "" {
		vers = "HTTP/1.1"
	}

	major, minor, ok := http.ParseHTTPVersion(vers)
	if ok {
		b.req.Proto = vers
		b.req.ProtoMajor = major
		b.req.ProtoMinor = minor
	}

	return b
}

// Cookie add cookie into request.
func (b *HTTPReq) Cookie(cookie *http.Cookie) *HTTPReq {
	b.req.Header.Add("Cookie", cookie.String())

	return b
}

// Transport set transport to
func (b *HTTPReq) Transport(transport http.RoundTripper) *HTTPReq {
	b.setting.Transport = transport

	return b
}

// Proxy set http proxy
// example:
//
//	func(req *http.Request) (*url.URL, error) {
// 		u, _ := url.ParseRequestURI("http://127.0.0.1:8118")
// 		return u, nil
// 	}
func (b *HTTPReq) Proxy(proxy func(*http.Request) (*url.URL, error)) *HTTPReq {
	b.setting.Proxy = proxy

	return b
}

// Param adds query param in to request.
// params build query string as ?key1=value1&key2=value2...
func (b *HTTPReq) Param(key, value string) *HTTPReq {
	b.params[key] = value

	return b
}

// PostFile ...
func (b *HTTPReq) PostFile(formName, filename string) *HTTPReq {
	b.files[formName] = filename

	return b
}

// Body adds request raw body.
// it supports string and []byte.
func (b *HTTPReq) Body(data interface{}) *HTTPReq {
	switch t := data.(type) {
	case string:
		bf := bytes.NewBufferString(t)
		b.req.Body = ioutil.NopCloser(bf)
		b.req.ContentLength = int64(len(t))
	case []byte:
		bf := bytes.NewBuffer(t)
		b.req.Body = ioutil.NopCloser(bf)
		b.req.ContentLength = int64(len(t))
	}

	return b
}

// JSONBody adds request raw body encoding by JSON.
func (b *HTTPReq) JSONBody(obj interface{}) error {
	if b.req.Body != nil || obj == nil {
		return errors.New("body should not be nil")
	}

	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)

	if err := enc.Encode(obj); err != nil {
		return err
	}

	b.req.Body = ioutil.NopCloser(buf)
	b.req.ContentLength = int64(buf.Len())
	b.req.Header.Set("Content-Type", "application/json;charset=utf-8")

	return nil
}

func (b *HTTPReq) buildURL(paramBody string) {
	// build GET url with query string
	m := b.req.Method
	if m == "GET" && paramBody != "" {
		if strings.Contains(b.url, "?") {
			b.url += "&" + paramBody
		} else {
			b.url += "?" + paramBody
		}

		return
	}

	// build POST/PUT/PATCH url and body
	if (m == "POST" || m == "PUT" || m == "PATCH") && b.req.Body == nil {
		// with files
		if len(b.files) > 0 {
			b.postFiles()

			return
		}

		// with params
		if len(paramBody) > 0 {
			b.Header("Content-Type", "application/x-www-form-urlencoded")
			b.Body(paramBody)
		}
	}
}

func (b *HTTPReq) postFiles() {
	pr, pw := io.Pipe()
	bodyWriter := multipart.NewWriter(pw)

	go func() {
		for formname, filename := range b.files {
			fileWriter, err := bodyWriter.CreateFormFile(formname, filename)
			if err != nil {
				log.Fatal(err)
			}

			fh, err := os.Open(filename)
			if err != nil {
				log.Fatal(err)
			}
			// iocopy
			_, err = io.Copy(fileWriter, fh)
			_ = fh.Close()

			if err != nil {
				log.Fatal(err)
			}
		}

		for k, v := range b.params {
			_ = bodyWriter.WriteField(k, v)
		}

		_ = bodyWriter.Close()
		_ = pw.Close()
	}()
	b.Header("Content-Type", bodyWriter.FormDataContentType())
	b.req.Body = ioutil.NopCloser(pr)
}

func (b *HTTPReq) getResponse() (*http.Response, error) {
	if b.resp.StatusCode != 0 {
		return b.resp, nil
	}

	resp, err := b.SendOut()

	if err != nil {
		return nil, err
	}

	b.resp = resp

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}

	return resp, errors.New(resp.Status)
}

// SendOut ...
func (b *HTTPReq) SendOut() (*http.Response, error) { // nolint funlen
	var paramBody string

	if len(b.params) > 0 {
		var buf bytes.Buffer

		for k, v := range b.params {
			buf.WriteString(url.QueryEscape(k))
			buf.WriteByte('=')
			buf.WriteString(url.QueryEscape(v))
			buf.WriteByte('&')
		}

		paramBody = buf.String()[0 : buf.Len()-1]
	}

	var err error

	b.buildURL(paramBody)

	if b.req.URL, err = url.Parse(b.url); err != nil {
		return nil, err
	}

	trans := b.setting.Transport
	if trans == nil {
		t := &http.Transport{TLSClientConfig: b.setting.TLSClientConfig,
			Proxy:       b.setting.Proxy,
			DialContext: TimeoutDialer(b.setting.ConnectTimeout, b.setting.ReadWriteTimeout),
		}

		defer t.CloseIdleConnections() // fd leak w/o this

		trans = t
	} else if t, ok := trans.(*http.Transport); ok {
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = b.setting.TLSClientConfig
		}

		if t.Proxy == nil {
			t.Proxy = b.setting.Proxy
		}

		if t.DialContext == nil {
			t.DialContext = TimeoutDialer(b.setting.ConnectTimeout, b.setting.ReadWriteTimeout)
		}
	}

	var jar http.CookieJar

	if b.setting.EnableCookie {
		if b.setting.CookieJar == nil {
			b.setting.CookieJar = NewCookieJar()
		}

		jar = b.setting.CookieJar
	}

	client := &http.Client{Transport: trans, Jar: jar}

	if b.setting.UserAgent != "" && b.req.Header.Get("User-Agent") == "" {
		b.req.Header.Set("User-Agent", b.setting.UserAgent)
	}

	if b.setting.ShowDebug {
		if b.dump, err = httputil.DumpRequest(b.req, b.setting.DumpBody); err != nil {
			log.Println(err.Error())
		}
	}

	return client.Do(b.req)
}

// String returns the body string in response.
// it calls Response inner.
func (b *HTTPReq) String() (string, error) {
	data, err := b.Bytes()
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Bytes returns the body []byte in response.
// it calls Response inner.
func (b *HTTPReq) Bytes() ([]byte, error) {
	if b.body != nil {
		return b.body, nil
	}

	resp, err := b.getResponse()

	if err != nil || resp.Body == nil {
		return nil, err
	}

	return b.ReadResponseBody(resp)
}

// ReadResponseBody ...
func (b *HTTPReq) ReadResponseBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()

	if b.setting.Gzip && resp.Header.Get("Content-Encoding") == "gzip" {
		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}

		return ioutil.ReadAll(reader)
	}

	return ioutil.ReadAll(resp.Body)
}

// ToFile saves the body data in response to one file.
// it calls Response inner.
func (b *HTTPReq) ToFile(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	resp, err := b.getResponse()
	if err != nil || resp.Body == nil {
		return err
	}

	defer resp.Body.Close()

	_, err = io.Copy(f, resp.Body)

	return err
}

// ToJSON returns the map that marshals from the body bytes as json in response .
// it calls Response inner.
func (b *HTTPReq) ToJSON(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}

// ToXML returns the map that marshals from the body bytes as xml in response .
// it calls Response inner.
func (b *HTTPReq) ToXML(v interface{}) error {
	data, err := b.Bytes()
	if err != nil {
		return err
	}

	return xml.Unmarshal(data, v)
}

// Response executes request client gets response manually.
func (b *HTTPReq) Response() (*http.Response, error) {
	return b.getResponse()
}

// Dialer defines dialer function alias
type Dialer func(ctx context.Context, net, addr string) (c net.Conn, err error)

// TimeoutDialer returns functions of connection dialer with timeout settings for http.Transport Dial field.
// https://gist.github.com/c4milo/275abc6eccbfd88ad56ca7c77947883a
// HTTP client with support for read and write timeouts which are missing in Go's standard library.
func TimeoutDialer(cTimeout time.Duration, rwTimeout time.Duration) Dialer {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(network, addr, cTimeout)
		if err != nil {
			return conn, err
		}

		if rwTimeout > 0 {
			err = conn.SetDeadline(time.Now().Add(rwTimeout))
		}

		return conn, err
	}
}
