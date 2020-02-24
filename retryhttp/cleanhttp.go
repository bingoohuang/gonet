package retryhttp

import (
	"net"
	"net/http"
	"runtime"
	"strings"
	"time"
	"unicode"
)

// DefaultTransport returns a new http.Transport with similar default values to
// http.DefaultTransport, but with idle connections and keepalives disabled.
func DefaultTransport() *http.Transport {
	t := DefaultPooledTransport()
	t.DisableKeepAlives = true
	t.MaxIdleConnsPerHost = -1

	return t
}

// DefaultPooledTransport returns a new http.Transport with similar default
// values to http.DefaultTransport. Do not use this for transient transports as
// it can leak file descriptors over time. Only use this for transports that
// will be re-used for the same host(s).
// nolint gomnd
func DefaultPooledTransport() *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,
	}
}

// DefaultClient returns a new http.Client with similar default values to
// http.Client, but with a non-shared Transport, idle connections disabled, and
// keepalives disabled.
func DefaultClient() *http.Client {
	return &http.Client{Transport: DefaultTransport()}
}

// DefaultPooledClient returns a new http.Client with similar default values to
// http.Client, but with a shared Transport. Do not use this function for
// transient clients as it can leak file descriptors over time. Only use this
// for clients that will be re-used for the same host(s).
func DefaultPooledClient() *http.Client {
	return &http.Client{Transport: DefaultPooledTransport()}
}

// HandlerInput provides input options to cleanhttp's handlers
type HandlerInput struct {
	ErrStatus int
}

// PrintablePathCheckHandler is a middleware that ensures the request path
// contains only printable runes.
func PrintablePathCheckHandler(next http.Handler, input *HandlerInput) http.Handler {
	// Nil-check on input to make it optional
	if input == nil {
		input = &HandlerInput{
			ErrStatus: http.StatusBadRequest,
		}
	}

	// Default to http.StatusBadRequest on error
	if input.ErrStatus == 0 {
		input.ErrStatus = http.StatusBadRequest
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			return
		}

		// Check URL path for non-printable characters
		idx := strings.IndexFunc(r.URL.Path, func(c rune) bool {
			return !unicode.IsPrint(c)
		})

		if idx != -1 {
			w.WriteHeader(input.ErrStatus)
			return
		}

		if next != nil {
			next.ServeHTTP(w, r)
		}
	})
}
