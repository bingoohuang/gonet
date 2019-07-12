# urllib

urllib is an libs help you to curl remote url using golang.
original from [urllib](https://github.com/GiterLab/urllib)

# How to use?

## Example

    import (
        "fmt"
        "github.com/bingoohuang/gonet"
    )
    
    func main() {
        req := gonet.MustGet("http://tobyzxj.me/").Debug(true)
        str, err := req.String()
        if err != nil {
            fmt.Println(err)
        }
        fmt.Println(req.DumpRequestString()) // debug out
        fmt.Println(str)
    }


## GET

you can use Get to crawl data.

    str, err := gonet.MustGet("http://tobyzxj.me/").String()
    if err != nil {
            // error
    }
    fmt.Println(str)
	
## POST

POST data to remote url

    req := MustPost("http://tobyzxj.me/")
    req.Param("username","tobyzxj")
    req.Param("password","123456")
    str, err := req.String()
    if err != nil {
            // error
    }
    fmt.Println(str)

## Set timeout

The default timeout is `10` seconds, function prototype:

	Timeout(connectTimeout, readWriteTimeout time.Duration)

Example:

	// GET
	MustGet("http://tobyzxj.me/").Timeout(100 * time.Second, 30 * time.Second)
	
	// POST
	MustPost("http://tobyzxj.me/").Timeout(100 * time.Second, 30 * time.Second)


## Debug

If you want to debug the request info, set the debug on

	MustGet("http://tobyzxj.me/").Debug(true)
	
## Set HTTP Basic Auth

	str, err := MustGet("http://tobyzxj.me/").BasicAuth("user", "passwd").String()
	if err != nil {
        	// error
	}
	fmt.Println(str)
	
## Set HTTPS

If request url is https, You can set the client support TSL:

	UrlSetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	
More info about the `tls.Config` please visit http://golang.org/pkg/crypto/tls/#Config	

## Set HTTP Version

some servers need to specify the protocol version of HTTP

	MustGet("http://tobyzxj.me/").SetProtocolVersion("HTTP/1.1")
	
## Set Cookie

some http request need setcookie. So set it like this:

	cookie := &http.Cookie{}
	cookie.Name = "username"
	cookie.Value  = "tobyzxj"
	MustGet("http://tobyzxj.me/").Cookie(cookie)

## Upload file

urllib support mutil file upload, use `req.PostFile()`

	req := MustPost("http://tobyzxj.me/")
	req.Param("username","tobyzxj")
	req.PostFile("uploadfile1", "Urlpdf")
	str, err := req.String()
	if err != nil {
        	// error
	}
	fmt.Println(str)
