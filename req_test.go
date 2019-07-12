package gonet

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"
)

func TestResponse(t *testing.T) {
	req := MustGet("http://httpbin.org/get")
	resp, err := req.Response()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)
}

func TestGet(t *testing.T) {
	req := MustGet("http://httpbin.org/get")
	s, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(s)
}

const smallfish = "smallfish"

func TestSimplePost(t *testing.T) {

	v := smallfish
	req := MustPost("http://httpbin.org/post")
	req.Param("username", v)

	str, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in post")
	}
}

func TestPostFile(t *testing.T) {
	v := smallfish
	req := MustPost("http://httpbin.org/post")
	req.Param("username", v)
	req.PostFile("uploadfile", "req_test.go")

	str, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in post")
	}
}

func TestSimplePut(t *testing.T) {
	str, err := MustPut("http://httpbin.org/put").String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
}

func TestSimpleDelete(t *testing.T) {
	str, err := MustDelete("http://httpbin.org/delete").String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
}

func TestWithCookie(t *testing.T) {
	v := smallfish
	jar := NewCookieJar()
	str, err := MustGet("http://httpbin.org/cookies/set?k1=" + v).CookieJar(jar).String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	str, err = MustGet("http://httpbin.org/cookies").CookieJar(jar).String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in cookie")
	}
}

func TestWithBasicAuth(t *testing.T) {
	str, err := MustGet("http://httpbin.org/basic-auth/user/passwd").BasicAuth("user", "passwd").String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
	n := strings.Index(str, "authenticated")
	if n == -1 {
		t.Fatal("authenticated not found in response")
	}
}

func TestWithUserAgent(t *testing.T) {
	v := "GiterLab"
	str, err := MustGet("http://httpbin.org/headers").UserAgent(v).String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in user-agent")
	}
}

func TestWithSetting(t *testing.T) {
	v := "Gonet"
	setting := NewReqOption()
	setting.EnableCookie = true
	setting.UserAgent = v
	setting.Transport = nil

	str, err := MustGet("http://httpbin.org/get").String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)

	n := strings.Index(str, v)
	if n == -1 {
		t.Fatal(v + " not found in user-agent")
	}
}

func TestToJson(t *testing.T) {
	req := MustGet("http://httpbin.org/ip")
	resp, err := req.Response()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(resp)

	// httpbin will return http remote addr
	type IP struct {
		Origin string `json:"origin"`
	}
	var ip IP
	err = req.ToJSON(&ip)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(ip.Origin)

	ips := strings.Split(ip.Origin, ",")

	for _, i := range ips {
		if n := strings.Count(i, "."); n != 3 {
			t.Fatal("response is not valid ip")
		}
	}
}

func TestToFile(t *testing.T) {
	f := "GiterLab_testfile"
	req := MustGet("http://httpbin.org/ip")
	err := req.ToFile(f)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f)
	b, err := ioutil.ReadFile(f)
	if n := strings.Index(string(b), "origin"); n == -1 {
		t.Fatal(err)
	}
}

func TestHeader(t *testing.T) {
	req := MustGet("http://httpbin.org/headers")
	req.Header("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_9_0) "+
		"AppleWebKit/537.36 (KHTML, like Gecko) Chrome/31.0.1650.57 Safari/537.36")
	str, err := req.String()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(str)
}
