retryhttp
================

copied from http://godoc.org/github.com/hashicorp/go-retryhttp.

The `retryhttp` package provides a familiar HTTP client interface with
automatic retries and exponential backoff. It is a thin wrapper over the
standard `net/http` client library and exposes nearly the same public API. This
makes `retryhttp` very easy to drop into existing programs.

`retryhttp` performs automatic retries under certain conditions. Mainly, if
an error is returned by the client (connection errors, etc.), or if a 500-range
response code is received (except 501), then a retry is invoked after a wait
period.  Otherwise, the response is returned and left to the caller to
interpret.

The main difference from `net/http` is that requests which take a request body
(POST/PUT et. al) can have the body provided in a number of ways (some more or
less efficient) that allow "rewinding" the request body if the initial request
fails so that the full request can be attempted again. 

Example Use
===========

Using this library should look almost identical to what you would do with
`net/http`. The most simple example of a GET request is shown below:

```go
resp, err := retryhttp.Get("/foo")
if err != nil {
    panic(err)
}
```

The returned response object is an `*http.Response`, the same thing you would
usually get from `net/http`. Had the request failed one or more times, the above
call would block and retry with exponential backoff.

Clean http
==========

Package retryhttp offers convenience utilities for acquiring "clean"
http.Transport and http.Client structs.

Values set on http.DefaultClient and http.DefaultTransport affect all
callers. This can have detrimental effects, esepcially in TLS contexts,
where client or root certificates set to talk to multiple endpoints can end
up displacing each other, leading to hard-to-debug issues. This package
provides non-shared http.Client and http.Transport structs to ensure that
the configuration will not be overwritten by other parts of the application
or dependencies.

The DefaultClient and DefaultTransport functions disable idle connections
and keepalives. Without ensuring that idle connections are closed before
garbage collection, short-term clients/transports can leak file descriptors,
eventually leading to "too many open files" errors. If you will be
connecting to the same hosts repeatedly from the same client, you can use
DefaultPooledClient to receive a client that has connection pooling
semantics similar to http.DefaultClient.
//