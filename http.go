package gonet

import (
	"bytes"
	"compress/gzip"
	"io"
	"mime"
	"net/http"
	"net/http/httputil"
	"os"
	"path/filepath"
	"strings"
)

type GzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
}

func (w GzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func GzipHandlerFn(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			fn(w, r)
			return
		}
		w.Header().Set("Content-Encoding", "gzip")
		gz := gzip.NewWriter(w)
		defer gz.Close()
		gzr := GzipResponseWriter{Writer: gz, ResponseWriter: w}
		fn(gzr, r)
	}
}

func DumpRequest(fn http.HandlerFunc, body bool, dumper func(error, []byte)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Save a copy of this request for debugging.
		requestDump, err := httputil.DumpRequest(r, body)
		dumper(err, requestDump)
		fn(w, r)
	}
}

func DetectContentType(name string) (t string) {
	if t = mime.TypeByExtension(filepath.Ext(name)); t == "" {
		t = "application/octet-stream"
	}
	return
}

func ServeImage(imageBytes []byte, fi os.FileInfo) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", DetectContentType(fi.Name()))
		w.Header().Set("Last-Modified", fi.ModTime().UTC().Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
		_, _ = io.Copy(w, bytes.NewReader(imageBytes))
	}
}

func ReadString(object io.ReadCloser) string {
	return string(ReadBytes(object))
}

func ReadBytes(object io.ReadCloser) []byte {
	defer object.Close()

	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(object)
	return buf.Bytes()
}

const ContentType = "Content-Type"

func ContentTypeHTML(w http.ResponseWriter) {
	w.Header().Set(ContentType, "text/html; charset=utf-8")
}

func ContentTypeJSON(w http.ResponseWriter) {
	w.Header().Set(ContentType, "application/json; charset=utf-8")
}