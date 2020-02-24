package retryhttp

import (
	"bytes"
	"context"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

// nolint gochecknoglobals
var (
	// Default retry configuration
	defaultRetryWaitMin = 1 * time.Second
	defaultRetryWaitMax = 30 * time.Second
	defaultRetryMax     = 4

	// defaultLogger is the logger provided with defaultClient
	defaultLogger = log.New(os.Stderr, "", log.LstdFlags)

	// defaultClient is used for performing requests without explicitly making
	// a new client. It is purposely private to avoid modifications.
	defaultClient = NewClient()

	// We need to consume response bodies to maintain http connections, but
	// limit the size we consume to respReadLimit.
	respReadLimit = int64(4096)

	// A regular expression to match the error returned by net/http when the
	// configured number of redirects is exhausted. This error isn't typed
	// specifically so we resort to matching on the error string.
	redirectsErrorRe = regexp.MustCompile(`stopped after \d+ redirects\z`)

	// A regular expression to match the error returned by net/http when the
	// scheme specified in the URL is invalid. This error isn't typed
	// specifically so we resort to matching on the error string.
	schemeErrorRe = regexp.MustCompile(`unsupported protocol scheme`)
)

// ReaderFunc is the type of function that can be given natively to NewRequest
type ReaderFunc func() (io.Reader, error)

// LenReader is an interface implemented by many in-memory io.Reader's. Used
// for automatically sending the right Content-Length header when possible.
type LenReader interface {
	Len() int
}

// Request wraps the metadata needed to create HTTP requests.
type Request struct {
	// body is a seekable reader over the request body payload. This is
	// used to rewind the request data in between retries.
	body ReaderFunc

	// Embed an HTTP request directly. This makes a *Request act exactly
	// like an *http.Request so that all meta methods are supported.
	*http.Request
}

// WithContext returns wrapped Request with a shallow copy of underlying *http.Request
// with its context changed to ctx. The provided ctx must be non-nil.
func (r *Request) WithContext(ctx context.Context) *Request {
	r.Request = r.Request.WithContext(ctx)
	return r
}

// BodyBytes allows accessing the request body. It is an analogue to
// http.Request's Body variable, but it returns a copy of the underlying data
// rather than consuming it.
//
// This function is not thread-safe; do not call it at the same time as another
// call, or at the same time this request is being used with Client.Do.
func (r *Request) BodyBytes() ([]byte, error) {
	if r.body == nil {
		return nil, nil
	}

	body, err := r.body()

	if err != nil {
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(body)

	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func dealReaderFunc(body ReaderFunc) (bodyReader ReaderFunc, contentLength int64, err error) {
	tmp, err := body()

	if err != nil {
		return nil, 0, err
	}

	if lr, ok := tmp.(LenReader); ok {
		contentLength = int64(lr.Len())
	}

	if c, ok := tmp.(io.Closer); ok {
		c.Close()
	}

	return body, contentLength, err
}

func dealBytes(body []byte) (bodyReader ReaderFunc, contentLength int64, err error) {
	buf := body
	bodyReader = func() (io.Reader, error) {
		return bytes.NewReader(buf), nil
	}
	contentLength = int64(len(buf))

	return bodyReader, contentLength, nil
}

func dealBuffer(body *bytes.Buffer) (bodyReader ReaderFunc, contentLength int64, err error) {
	buf := body
	bodyReader = func() (io.Reader, error) {
		return bytes.NewReader(buf.Bytes()), nil
	}
	contentLength = int64(buf.Len())

	return bodyReader, contentLength, nil
}

func dealReader(body *bytes.Reader) (bodyReader ReaderFunc, contentLength int64, err error) {
	buf, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, 0, err
	}

	bodyReader = func() (io.Reader, error) { return bytes.NewReader(buf), nil }
	contentLength = int64(len(buf))

	return bodyReader, contentLength, nil
}

func dealReadSeeker(body io.ReadSeeker) (bodyReader ReaderFunc, contentLength int64, err error) {
	raw := body
	bodyReader = func() (io.Reader, error) {
		_, err := raw.Seek(0, 0)
		return ioutil.NopCloser(raw), err
	}

	if lr, ok := raw.(LenReader); ok {
		contentLength = int64(lr.Len())
	}

	return bodyReader, contentLength, nil
}

func dealIOReader(body io.Reader) (bodyReader ReaderFunc, contentLength int64, err error) {
	buf, err := ioutil.ReadAll(body)
	if err != nil {
		return nil, 0, err
	}

	bodyReader = func() (io.Reader, error) { return bytes.NewReader(buf), nil }
	contentLength = int64(len(buf))

	return bodyReader, contentLength, nil
}

func getBodyReaderAndContentLength(rawBody interface{}) (ReaderFunc, int64, error) {
	if rawBody == nil {
		return nil, 0, nil
	}

	switch body := rawBody.(type) { // If they gave us a function already, great! Use it.
	case ReaderFunc:
		return dealReaderFunc(body)
	case func() (io.Reader, error):
		return dealReaderFunc(body)

	case []byte: // If a regular byte slice, we can read it over and over via new readers
		return dealBytes(body)
	case *bytes.Buffer: // If a bytes.Buffer we can read the underlying byte slice over and over
		return dealBuffer(body)

	// We prioritize *bytes.Reader here because we don't really want to
	// deal with it seeking so want it to match here instead of the io.ReadSeeker case.
	case *bytes.Reader:
		return dealReader(body)
	case io.ReadSeeker: // Compat case
		return dealReadSeeker(body)
	case io.Reader: // Read all in so we can reset
		return dealIOReader(body)
	default:
		return nil, 0, fmt.Errorf("cannot handle type %T", rawBody)
	}
}

// FromRequest wraps an http.Request in a retryhttp.Request
func FromRequest(r *http.Request) (*Request, error) {
	bodyReader, _, err := getBodyReaderAndContentLength(r.Body)
	if err != nil {
		return nil, err
	}
	// Could assert contentLength == r.ContentLength
	return &Request{bodyReader, r}, nil
}

// NewRequest creates a new wrapped request.
func NewRequest(method, url string, rawBody interface{}) (*Request, error) {
	bodyReader, contentLength, err := getBodyReaderAndContentLength(rawBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}

	httpReq.ContentLength = contentLength

	return &Request{bodyReader, httpReq}, nil
}

// Logger interface allows to use other loggers than
// standard log.Logger.
type Logger interface {
	Printf(string, ...interface{})
}

// LeveledLogger interface implements the basic methods that a logger library needs
type LeveledLogger interface {
	Errorf(string, ...interface{})
	Infof(string, ...interface{})
	Debugf(string, ...interface{})
	Warnf(string, ...interface{})
}

// hookLogger adapts an LeveledLogger to Logger for use by the existing hook functions
// without changing the API.
type hookLogger struct {
	LeveledLogger
}

func (h hookLogger) Printf(s string, args ...interface{}) {
	h.Infof(s, args...)
}

// RequestLogHook allows a function to run before each retry. The HTTP
// request which will be made, and the retry number (0 for the initial
// request) are available to users. The internal logger is exposed to
// consumers.
type RequestLogHook func(Logger, *http.Request, int)

// ResponseLogHook is like RequestLogHook, but allows running a function
// on each HTTP response. This function will be invoked at the end of
// every HTTP request executed, regardless of whether a subsequent retry
// needs to be performed or not. If the response body is read or closed
// from this method, this will affect the response returned from Do().
type ResponseLogHook func(Logger, *http.Response)

// CheckRetry specifies a policy for handling retries. It is called
// following each request with the response and error values returned by
// the http.Client. If CheckRetry returns false, the Client stops retrying
// and returns the response to the caller. If CheckRetry returns an error,
// that error value is returned in lieu of the error from the request. The
// Client will close any response body when retrying, but if the retry is
// aborted it is up to the CheckRetry callback to properly close any
// response body before returning.
type CheckRetry func(ctx context.Context, resp *http.Response, err error) (bool, error)

// Backoff specifies a policy for how long to wait between retries.
// It is called after a failing request to determine the amount of time
// that should pass before trying again.
type Backoff func(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration

// ErrorHandler is called if retries are expired, containing the last status
// from the http library. If not specified, default behavior for the library is
// to close the body and return an error indicating how many tries were
// attempted. If overriding this, be sure to close the body if needed.
type ErrorHandler func(resp *http.Response, err error, numTries int) (*http.Response, error)

// Client is used to make HTTP requests. It adds additional functionality
// like automatic retries to tolerate minor outages.
type Client struct {
	HTTPClient *http.Client // Internal HTTP client.
	Logger     interface{}  // Customer logger instance. Can be either Logger or LeveledLogger

	RetryWaitMin time.Duration // Minimum time to wait
	RetryWaitMax time.Duration // Maximum time to wait
	RetryMax     int           // Maximum number of retries

	// RequestLogHook allows a user-supplied function to be called
	// before each retry.
	RequestLogHook RequestLogHook

	// ResponseLogHook allows a user-supplied function to be called
	// with the response from each HTTP request executed.
	ResponseLogHook ResponseLogHook

	// CheckRetry specifies the policy for handling retries, and is called
	// after each request. The default policy is DefaultRetryPolicy.
	CheckRetry CheckRetry

	// Backoff specifies the policy for how long to wait between retries
	Backoff Backoff

	// ErrorHandler specifies the custom error handler to use, if any
	ErrorHandler ErrorHandler

	loggerInit sync.Once
}

// NewClient creates a new Client with default settings.
func NewClient() *Client {
	return &Client{
		HTTPClient:   DefaultPooledClient(),
		Logger:       defaultLogger,
		RetryWaitMin: defaultRetryWaitMin,
		RetryWaitMax: defaultRetryWaitMax,
		RetryMax:     defaultRetryMax,
		CheckRetry:   DefaultRetryPolicy,
		Backoff:      DefaultBackoff,
	}
}

func (c *Client) logger() interface{} {
	c.loggerInit.Do(func() {
		if c.Logger == nil {
			return
		}

		switch c.Logger.(type) {
		case Logger, LeveledLogger:
			// ok
		default:
			// This should happen in dev when they are setting Logger and work on code, not in prod.
			panic(fmt.Sprintf("invalid logger type passed, must be Logger or LeveledLogger, was %T", c.Logger))
		}
	})

	return c.Logger
}

// DefaultRetryPolicy provides a default callback for Client.CheckRetry, which
// will retry on connection errors and server errors.
func DefaultRetryPolicy(ctx context.Context, resp *http.Response, err error) (bool, error) {
	// do not retry on context.Canceled or context.DeadlineExceeded
	if ctx.Err() != nil {
		return false, ctx.Err()
	}

	if err != nil {
		if v, ok := err.(*url.Error); ok {
			// Don't retry if the error was due to too many redirects.
			if redirectsErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to an invalid protocol scheme.
			if schemeErrorRe.MatchString(v.Error()) {
				return false, nil
			}

			// Don't retry if the error was due to TLS cert verification failure.
			if _, ok := v.Err.(x509.UnknownAuthorityError); ok {
				return false, nil
			}
		}

		// The error is likely recoverable so retry.
		return true, nil
	}

	// Check the response code. We retry on 500-range responses to allow
	// the server time to recover, as 500's are typically not permanent
	// errors and may relate to outages on the server side. This will catch
	// invalid response codes as well, like 0 and 999.
	if resp.StatusCode == 0 || (resp.StatusCode >= 500 && resp.StatusCode != 501) {
		return true, nil
	}

	return false, nil
}

// DefaultBackoff provides a default callback for Client.Backoff which
// will perform exponential backoff based on the attempt number and limited
// by the provided minimum and maximum durations.
func DefaultBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	mult := math.Pow(2, float64(attemptNum)) * float64(min) // nolint gomnd
	sleep := time.Duration(mult)

	if float64(sleep) != mult || sleep > max {
		sleep = max
	}

	return sleep
}

// LinearJitterBackoff provides a callback for Client.Backoff which will
// perform linear backoff based on the attempt number and with jitter to
// prevent a thundering herd.
//
// min and max here are *not* absolute values. The number to be multiplied by
// the attempt number will be chosen at random from between them, thus they are
// bounding the jitter.
//
// For instance:
// * To get strictly linear backoff of one second increasing each retry, set
// both to one second (1s, 2s, 3s, 4s, ...)
// * To get a small amount of jitter centered around one second increasing each
// retry, set to around one second, such as a min of 800ms and max of 1200ms
// (892ms, 2102ms, 2945ms, 4312ms, ...)
// * To get extreme jitter, set to a very wide spread, such as a min of 100ms
// and a max of 20s (15382ms, 292ms, 51321ms, 35234ms, ...)
func LinearJitterBackoff(min, max time.Duration, attemptNum int, resp *http.Response) time.Duration {
	// attemptNum always starts at zero but we want to start at 1 for multiplication
	attemptNum++

	if max <= min {
		// Unclear what to do here, or they are the same, so return min *
		// attemptNum
		return min * time.Duration(attemptNum)
	}

	// Seed rand; doing this every time is fine
	rand := rand.New(rand.NewSource(int64(time.Now().Nanosecond())))

	// Pick a random number that lies somewhere between the min and max and
	// multiply by the attemptNum. attemptNum starts at zero so we always
	// increment here. We first get a random percentage, then apply that to the
	// difference between min and max, and add to min.
	jitter := rand.Float64() * float64(max-min)
	jitterMin := int64(jitter) + int64(min)

	return time.Duration(jitterMin * int64(attemptNum))
}

// PassthroughErrorHandler is an ErrorHandler that directly passes through the
// values from the net/http library for the final request. The body is not
// closed.
func PassthroughErrorHandler(resp *http.Response, err error, _ int) (*http.Response, error) {
	return resp, err
}

func (c *Client) debugf(format string, args ...interface{}) {
	logger := c.logger()

	if logger == nil {
		return
	}

	switch v := logger.(type) {
	case Logger:
		v.Printf(format, args...)
	case LeveledLogger:
		v.Debugf(format, args...)
	}
}

func (c *Client) errorf(format string, args ...interface{}) {
	logger := c.logger()

	if logger == nil {
		return
	}

	switch v := logger.(type) {
	case Logger:
		v.Printf(format, args...)
	case LeveledLogger:
		v.Errorf(format, args...)
	}
}

// Do wraps calling an HTTP method with retries.
func (c *Client) Do(req *Request) (resp *http.Response, err error) {
	if c.HTTPClient == nil {
		c.HTTPClient = DefaultPooledClient()
	}

	defer c.HTTPClient.CloseIdleConnections()

	c.debugf("[DEBUG] %s %s", req.Method, req.URL)

	logger := c.parseLogger()

	var n next

LOOP:
	for i := 0; ; i++ {
		switch n, resp, err = c.retry(req, logger, i); n {
		case errorReturn:
			return resp, err
		case continueLoop:
			continue
		case breakLoop:
			break LOOP
		}
	}

	if c.ErrorHandler != nil {
		return c.ErrorHandler(resp, err, c.RetryMax+1) // nolint gomnd
	}

	// By default, we close the response body and return an error without
	// returning the response
	if resp != nil {
		resp.Body.Close()
	}

	return nil, fmt.Errorf("%s %s giving up after %d attempts",
		req.Method, req.URL, c.RetryMax+1) // nolint gomnd
}

type next int

const (
	errorReturn next = iota
	breakLoop
	continueLoop
)

func (c *Client) retry(req *Request, logger Logger, i int) (n next, resp *http.Response, err error) {
	if err := c.rewindRequestBody(req); err != nil {
		return errorReturn, nil, err
	}

	if c.RequestLogHook != nil {
		c.RequestLogHook(logger, req.Request, i)
	}

	code := 0 // HTTP response code
	// Attempt the request
	resp, err = c.HTTPClient.Do(req.Request)
	if resp != nil {
		code = resp.StatusCode
	}

	// Check if we should continue with retries.
	checkOK, checkErr := c.CheckRetry(req.Context(), resp, err)

	if err != nil {
		c.errorf("[ERR] %s %s request failed: %v", req.Method, req.URL, err)
	} else if c.ResponseLogHook != nil {
		// Call this here to maintain the behavior of logging all requests,
		// even if CheckRetry signals to stop.

		// Call the response logger function if provided.
		c.ResponseLogHook(logger, resp)
	}

	// Now decide if we should continue.
	if !checkOK {
		if checkErr != nil {
			err = checkErr
		}

		return errorReturn, resp, err
	}

	// We do this before drainBody beause there's no need for the I/O if
	// we're breaking out
	remain := c.RetryMax - i
	if remain <= 0 {
		return breakLoop, resp, err
	}

	// We're going to retry, consume any response to reuse the connection.
	if err == nil && resp != nil {
		c.drainBody(resp.Body)
	}

	wait := c.Backoff(c.RetryWaitMin, c.RetryWaitMax, i, resp)
	desc := fmt.Sprintf("%s %s (status: %d)", req.Method, req.URL, code)

	c.debugf("[DEBUG] %s: retrying in %s (%d left)", desc, wait, remain)

	if done, err := c.waitDone(req, wait); done {
		return errorReturn, resp, err
	}

	return continueLoop, resp, nil
}

func (c *Client) waitDone(req *Request, wait time.Duration) (bool, error) {
	select {
	case <-req.Context().Done():
		c.HTTPClient.CloseIdleConnections()
		return true, req.Context().Err()
	case <-time.After(wait):
	}

	return false, nil
}

func (c *Client) rewindRequestBody(req *Request) error {
	if req.body == nil {
		return nil
	}

	// Always rewind the request body when non-nil.
	body, err := req.body()
	if err != nil {
		c.HTTPClient.CloseIdleConnections()
		return err
	}

	if c, ok := body.(io.ReadCloser); ok {
		req.Body = c
	} else {
		req.Body = ioutil.NopCloser(body)
	}

	return nil
}

func (c *Client) parseLogger() Logger {
	switch v := c.logger().(type) {
	case Logger:
		return v
	case LeveledLogger:
		return hookLogger{v}
	default:
		return nil
	}
}

// Try to read the response body so we can reuse this connection.
func (c *Client) drainBody(body io.ReadCloser) {
	defer body.Close()
	_, err := io.Copy(ioutil.Discard, io.LimitReader(body, respReadLimit))

	if err != nil {
		c.errorf("[ERR] error reading response body: %v", err)
	}
}

// Get is a shortcut for doing a GET request without making a new client.
func Get(url string) (*http.Response, error) {
	return defaultClient.Get(url)
}

// Get is a convenience helper for doing simple GET requests.
func (c *Client) Get(url string) (*http.Response, error) {
	req, err := NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// Head is a shortcut for doing a HEAD request without making a new client.
func Head(url string) (*http.Response, error) {
	return defaultClient.Head(url)
}

// Head is a convenience method for doing simple HEAD requests.
func (c *Client) Head(url string) (*http.Response, error) {
	req, err := NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// Post is a shortcut for doing a POST request without making a new client.
func Post(url, bodyType string, body interface{}) (*http.Response, error) {
	return defaultClient.Post(url, bodyType, body)
}

// Post is a convenience method for doing simple POST requests.
func (c *Client) Post(url, bodyType string, body interface{}) (*http.Response, error) {
	req, err := NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", bodyType)

	return c.Do(req)
}

// PostForm is a shortcut to perform a POST with form data without creating
// a new client.
func PostForm(url string, data url.Values) (*http.Response, error) {
	return defaultClient.PostForm(url, data)
}

// PostForm is a convenience method for doing simple POST operations using
// pre-filled url.Values form data.
func (c *Client) PostForm(url string, data url.Values) (*http.Response, error) {
	return c.Post(url, "application/x-www-form-urlencoded", strings.NewReader(data.Encode()))
}
