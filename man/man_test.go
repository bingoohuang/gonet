package man_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/bingoohuang/gonet/man"

	"github.com/bingoohuang/gonet/tlsconf"

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
	man.T `timeout:"5s" method:"POST"`

	AddAgent func(man.URL, Agent) Result `dump:"req,rsp"`
}

// nolint gochecknoglobals
var man1 = func() (p poster1) { man.New(&p); return }()

func TestMan1(t *testing.T) {
	agent := Agent{Name: "bingoo", Age: 100}
	result := Result{State: 0, Message: "OK"}
	method := ""

	var requestAgent Agent

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_ = json.Unmarshal(gonet.ReadBytes(r.Body), &requestAgent)

		w.Header().Set(gonet.ContentType, man.HeadJSON)

		jv, _ := json.Marshal(result)
		_, _ = w.Write(jv)
	}))
	defer ts.Close()

	result2 := man1.AddAgent(man.URL(ts.URL), agent)

	assert.Equal(t, result, result2)
	assert.Equal(t, "POST", method)
	assert.Equal(t, agent, requestAgent)
}

type Poster2 struct {
	man.URL

	AddAgent func(Agent) Result
}

// nolint gochecknoglobals
var man2 = func() *Poster2 { p := new(Poster2); man.New(p); return p }()

func TestMan2(t *testing.T) {
	defer noPanic(t)

	agent := Agent{Name: "bingoo", Age: 100}
	result := Result{State: 0, Message: "OK"}
	method := ""

	var requestAgent Agent

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		_ = json.Unmarshal(gonet.ReadBytes(r.Body), &requestAgent)

		w.Header().Set(gonet.ContentType, man.HeadJSON)

		jv, _ := json.Marshal(result)
		_, _ = w.Write(jv)
	}))
	defer ts.Close()

	man2.URL = man.URL(ts.URL)
	result2 := man2.AddAgent(agent)

	assert.Equal(t, result, result2)
	assert.Equal(t, "GET", method)
	assert.Equal(t, agent, requestAgent)
}

type Poster3 struct {
	man.T `method:"POST"`

	Upload  func(man.URL, man.UploadFile) Result
	Upload2 func(man.URL, man.UploadFile, map[string]string) Result
}

func noPanic(t *testing.T) {
	if r := recover(); r != nil {
		t.Errorf("The code did not panic")
	}
}

// nolint gochecknoglobals
var man3 = func() (p Poster3) { man.New(&p); return }()

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

		w.Header().Set(gonet.ContentType, man.HeadJSON)

		jv, _ := json.Marshal(result)
		_, _ = w.Write(jv)
	}))
	defer ts.Close()

	f, _ := os.Open("testdata/upload.txt")
	result2 := man3.Upload(man.URL(ts.URL), man.MakeFile("file", "upload.txt", f))

	f.Close()

	assert.Equal(t, result, result2)
	assert.Empty(t, value)
	assert.Equal(t, "POST", method)
	assert.Equal(t, "upload.txt", filename)
	assert.Equal(t, []byte("bingoohuang"), filebytes)

	f, _ = os.Open("testdata/upload.txt")
	result2 = man3.Upload2(man.URL(ts.URL), man.MakeFile("file", "upload.txt", f),
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
	Download func(man.URL, *man.DownloadFile)
}

// nolint gochecknoglobals
var man4 = func() (p Poster4) { man.New(&p); return }()

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

	df := &man.DownloadFile{Writer: &buf}
	man4.Download(man.URL(ts.URL), df)

	assert.Equal(t, "upload.txt", df.Filename)
	assert.Equal(t, "bingoohuang", buf.String())
}

type Poster5 struct {
	Download func(man.URL, *man.DownloadFile) error
}

// nolint gochecknoglobals
var man5 = func() (p Poster5) { man.New(&p); return }()

func TestMan5(t *testing.T) {
	err := man5.Download("http://127.0.0.1:8123", nil)
	assert.NotNil(t, err)

	e, ok := err.(*url.Error)
	assert.True(t, ok)
	assert.NotNil(t, e)
}

func TestQuery(t *testing.T) {
	u := man.QueryURL("http://a.b.c", "k", "v", "k2", "v2")
	assert.Equal(t, man.URL("http://a.b.c?k=v&k2=v2"), u)

	u = man.QueryURL("http://a.b.c?a=b", "k", "v", "k2")
	assert.Equal(t, man.URL("http://a.b.c?a=b&k=v&k2="), u)

	u = man.QueryURL("http://a.b.c?a=b", "k", " ", "k2", "黄进兵")
	assert.Equal(t, man.URL("http://a.b.c?a=b&k=+&k2=%E9%BB%84%E8%BF%9B%E5%85%B5"), u)
}

type Poster6 struct {
	Hello          func(man.URL, man.TLSConfDir) string `tlsConfFiles:"client.key,client.pem,root.pem"`
	HelloUntrusted func(man.URL) error
}

// nolint gochecknoglobals
var man6 = func() (p Poster6) { man.New(&p); return }()

type Poster7 struct {
	man.T `tlsConfFiles:"client.key,client.pem,root.pem"`

	Hello func(man.URL, man.TLSConfDir) string
}

// nolint gochecknoglobals
var man7 = func() (p Poster7) { man.New(&p); return }()

func TestHttps6(t *testing.T) {
	dir, err := ioutil.TempDir("", "man")
	assert.Nil(t, err)

	defer os.RemoveAll(dir)

	filepath.Join()

	assert.Nil(t, tlsconf.TLSGenRootFiles(dir, "root.key", "root.pem"))
	assert.Nil(t, tlsconf.TLSGenServerFiles(dir, "root.key", "root.pem", "",
		"server.key", "server.pem"))
	assert.Nil(t, tlsconf.TLSGenClientFiles(dir, "root.key", "root.pem",
		"client.key", "client.pem"))

	ts := tlsconf.NewHTTPSTestServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(gonet.ContentType, "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("bingoohuang"))
	}), filepath.Join(dir, "server.pem"), filepath.Join(dir, "server.key"), filepath.Join(dir, "root.pem"))

	defer ts.Close()

	assert.Equal(t, "bingoohuang", man6.Hello(man.URL(ts.URL), man.TLSConfDir(dir)))
	assert.Error(t, man6.HelloUntrusted(man.URL(ts.URL)))

	assert.Equal(t, "bingoohuang", man7.Hello(man.URL(ts.URL), man.TLSConfDir(dir)))
}
