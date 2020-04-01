package gonet_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/bingoohuang/gonet"
	"github.com/stretchr/testify/assert"
)

type Agent struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Result struct {
	State   int    `json:"state"`
	Message string `json:"message"`
}

type poster1 struct {
	gonet.T `timeout:"5s" method:"POST"`

	AddAgent func(gonet.URL, Agent) Result `dump:"req,rsp"`
}

// nolint gochecknoglobals
var man1 = func() *poster1 {
	p := &poster1{}
	if err := gonet.NewMan(p); err != nil {
		panic(err)
	}

	return p
}()

func TestMan1(t *testing.T) {
	agent := Agent{Name: "bingoo", Age: 100}
	result := Result{State: 0, Message: "OK"}
	method := ""

	var requestAgent Agent

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_ = json.Unmarshal(gonet.ReadBytes(r.Body), &requestAgent)

		w.Header().Set(gonet.ContentType, gonet.HeadJSON)

		jv, _ := json.Marshal(result)
		_, _ = w.Write(jv)
	}))
	defer ts.Close()

	result2 := man1.AddAgent(gonet.URL(ts.URL), agent)

	assert.Equal(t, result, result2)
	assert.Equal(t, "POST", method)
	assert.Equal(t, agent, requestAgent)
}

type Poster2 struct {
	gonet.URL

	AddAgent func(Agent) Result
}

// nolint gochecknoglobals
var man2 = func() *Poster2 {
	p := &Poster2{}
	if err := gonet.NewMan(p); err != nil {
		panic(err)
	}

	return p
}()

func TestMan2(t *testing.T) {
	agent := Agent{Name: "bingoo", Age: 100}
	result := Result{State: 0, Message: "OK"}
	method := ""

	var requestAgent Agent

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_ = json.Unmarshal(gonet.ReadBytes(r.Body), &requestAgent)

		w.Header().Set(gonet.ContentType, gonet.HeadJSON)

		jv, _ := json.Marshal(result)
		_, _ = w.Write(jv)
	}))
	defer ts.Close()

	man2.URL = gonet.URL(ts.URL)
	result2 := man2.AddAgent(agent)

	assert.Equal(t, result, result2)
	assert.Equal(t, "GET", method)
	assert.Equal(t, agent, requestAgent)
}

type Poster3 struct {
	gonet.T `method:"POST"`

	Upload  func(gonet.URL, gonet.UploadFile) Result
	Upload2 func(gonet.URL, gonet.UploadFile, map[string]string) Result
}

// nolint gochecknoglobals
var man3 = func() *Poster3 {
	p := &Poster3{}
	if err := gonet.NewMan(p); err != nil {
		panic(err)
	}

	return p
}()

func TestMan3(t *testing.T) {
	result := Result{State: 0, Message: "OK"}
	method := ""
	filename := ""
	value := ""

	var filebytes []byte

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		filename, filebytes, _ = ReceiveFile(r, "file")
		value = r.FormValue("key")

		w.Header().Set(gonet.ContentType, gonet.HeadJSON)

		jv, _ := json.Marshal(result)
		_, _ = w.Write(jv)
	}))
	defer ts.Close()

	f, _ := os.Open("testdata/upload.txt")
	result2 := man3.Upload(gonet.URL(ts.URL), gonet.MakeFile("file", "upload.txt", f))

	f.Close()

	assert.Equal(t, result, result2)
	assert.Empty(t, value)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "upload.txt", filename)
	assert.Equal(t, []byte("bingoohuang"), filebytes)

	f, _ = os.Open("testdata/upload.txt")
	result2 = man3.Upload2(gonet.URL(ts.URL), gonet.MakeFile("file", "upload.txt", f),
		map[string]string{"key": "value"})

	f.Close()

	assert.Equal(t, result, result2)
	assert.Equal(t, "value", value)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "upload.txt", filename)
	assert.Equal(t, []byte("bingoohuang"), filebytes)
}

func ReceiveFile(r *http.Request, filenameKey string) (string, []byte, error) {
	_ = r.ParseMultipartForm(32 << 20) // limit your max input length!

	file, header, err := r.FormFile(filenameKey)
	if err != nil {
		return "", nil, err
	}

	defer file.Close()

	var buf bytes.Buffer

	if _, err := io.Copy(&buf, file); err != nil {
		return "", nil, err
	}

	return header.Filename, buf.Bytes(), nil
}

type Poster4 struct {
	Download func(gonet.URL, *gonet.DownloadFile)
}

// nolint gochecknoglobals
var man4 = func() *Poster4 {
	p := &Poster4{}
	if err := gonet.NewMan(p); err != nil {
		panic(err)
	}

	return p
}()

func TestMan4(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// https://golangtc.com/t/54d9ca47421aa9170200000f
		h := w.Header().Set
		h("Content-Disposition", "attachment; filename="+url.QueryEscape("upload.txt"))
		h("Content-Description", "File Transfer")
		h("Content-Type", "application/octet-stream")
		h("Content-Transfer-Encoding", "binary")
		h("Expires", "0")
		h("Cache-Control", "must-revalidate")
		h("Pragma", "public")

		http.ServeFile(w, r, "testdata/upload.txt")
	}))

	defer ts.Close()

	var buf bytes.Buffer

	df := &gonet.DownloadFile{Writer: &buf}
	man4.Download(gonet.URL(ts.URL), df)

	assert.Equal(t, "upload.txt", df.Filename)
	assert.Equal(t, "bingoohuang", buf.String())
}

type Poster5 struct {
	Download func(gonet.URL, *gonet.DownloadFile) error
}

// nolint gochecknoglobals
var man5 = func() *Poster5 {
	p := &Poster5{}
	if err := gonet.NewMan(p); err != nil {
		panic(err)
	}

	return p
}()

func TestMan5(t *testing.T) {
	err := man5.Download("http://127.0.0.1:8123", nil)
	assert.NotNil(t, err)

	e, ok := err.(*url.Error)
	assert.True(t, ok)
	assert.NotNil(t, e)
}

type Poster6 struct {
	Error error

	Download func(gonet.URL, *gonet.DownloadFile)
}

// nolint gochecknoglobals
var man6 = func() *Poster6 {
	p := &Poster6{}
	if err := gonet.NewMan(p); err != nil {
		panic(err)
	}

	return p
}()

func TestMan6(t *testing.T) {
	man6.Download("http://127.0.0.1:8123", nil)
	assert.NotNil(t, man6.Error)

	e, ok := man6.Error.(*url.Error)
	assert.True(t, ok)
	assert.NotNil(t, e)
}
