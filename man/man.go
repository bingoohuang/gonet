package man

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/bingoohuang/gonet"

	"github.com/bingoohuang/gonet/tlsconf"

	"github.com/sirupsen/logrus"

	"github.com/bingoohuang/gor"
)

// T is the a special interface for additional tag for the man1.
type T struct{}

// URL is the URL address for the http requests.
type URL string

// Method is the HTTP Method for the http requests.
type Method string

// Timeout is the timeout setting for the http requests.
type Timeout string

// Keepalive is the keep-alive flag (true(on,yes,1) /false(no, off,0), default true.
type Keepalive string

const (
	// HeadJSON is the const content type of JSON.
	HeadJSON = "application/json; charset=utf-8"
)

// UploadFile represents the upload file to be uploaded.
type UploadFile struct {
	FilenameKey string
	Filename    string
	Reader      io.Reader
}

// DownloadFile represents the downloaded file.
type DownloadFile struct {
	Filename string
	Writer   io.Writer
}

// MakeFile makes a file to upload.
func MakeFile(filenameKey, filename string, reader io.Reader) UploadFile {
	return UploadFile{FilenameKey: filenameKey, Filename: filename, Reader: reader}
}

// TLSConfFiles specifies the TLS configuration files for the client.
// like client.key,client.pem,root.pem
type TLSConfFiles string

// TLSConfDir specifies the TLSFiles configuration files directly for the client.
// like client.key,client.pem,root.pem
type TLSConfDir string

// Option is the options for Man.
type Option struct {
	// URL ...
	URL string

	urlField reflect.Value

	// Method ...
	Method string
	// Keepalive ...
	Keepalive string
	// Timeout ...
	Timeout string
	// TLSConfFiles like clientKeyFile,clientCertFile,serverRootCA(required=false)
	TLSConfFiles string
	TLSConfDir   string

	ErrSetter func(err error)
	Logger    Logger

	Client HTTPClient
}

// WithClient specifies the http client for the man.
func WithClient(c HTTPClient) OptionFn { return func(o *Option) { o.Client = c } }

// OptionFn is the func prototype for Option.
type OptionFn func(*Option)

// New makes a new Man for http requests.
func New(man interface{}, optionFns ...OptionFn) {
	if err := NewE(man, optionFns...); err != nil {
		panic(err)
	}
}

// New makes a new Man for http requests.
func NewE(man interface{}, optionFns ...OptionFn) error {
	v := reflect.ValueOf(man)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("man1 shoud be a pointer")
	}

	v = v.Elem()

	structValue := MakeStructValue(v)
	option := makeOption(structValue, v, optionFns)

	for i := 0; i < structValue.NumField; i++ {
		f := structValue.FieldByIndex(i)
		if f.PkgPath != "" || f.Kind != reflect.Func {
			continue
		}

		if err := createFn(option, f); err != nil {
			return err
		}
	}

	return nil
}

func createFn(option *Option, f StructField) error {
	numIn := f.Type.NumIn()
	numOut := f.Type.NumOut()

	lastOutError := numOut > 0 && gor.IsError(f.Type.Out(numOut-1)) // nolint gomnd
	if lastOutError {
		numOut--
	}

	fn := makeFunc(option, f, numIn, numOut)
	if fn == nil {
		return fmt.Errorf("unsupportd func %s %v", f.Name, f.Type)
	}

	f.Field.Set(reflect.MakeFunc(f.Type, func(args []reflect.Value) []reflect.Value {
		option.ErrSetter(nil)

		values, err := fn(args)
		if err != nil {
			option.ErrSetter(err)
			option.Logger.LogError(err)

			values = make([]reflect.Value, numOut, numOut+1) // nolint gomnd

			for i := 0; i < numOut; i++ {
				values[i] = reflect.Zero(f.Type.Out(i))
			}
		}

		if lastOutError {
			if err != nil {
				values = append(values, reflect.ValueOf(err))
			} else {
				values = append(values, reflect.Zero(gor.ErrType))
			}
		}

		return values
	}))

	return nil
}

type generalFn func(args []reflect.Value) ([]reflect.Value, error)

type runner struct {
	method                   string
	tlsConfDir, tlsConfFiles string
	dumpOption               string
	inputs                   []reflect.Value
	timeout                  time.Duration
	addr                     string
	keepalive                string
	option                   *Option
	httpClient               HTTPClient
}

func newRunner(option *Option, f StructField, numIn int, args []reflect.Value) (r *runner, err error) {
	r = &runner{option: option}

	timeout := gotOption(timeoutType, "timeout", option.Timeout, f, numIn, args)

	if r.timeout, err = time.ParseDuration(timeout); err != nil {
		return nil, fmt.Errorf("failed to parse timeout %s error: %v", timeout, err)
	}

	if r.addr, err = parseURL(args, option, f, numIn); err != nil {
		return nil, err
	}

	r.method = gotOption(methodType, "method", option.Method, f, numIn, args)
	r.dumpOption = gotOption(nil, "dump", option.Method, f, numIn, args)
	r.inputs = gotInputs(f, numIn, args)

	switch httpClientValue := findArgsImpl(f, numIn, args, httpClientType); {
	case httpClientValue.IsValid():
		r.httpClient = httpClientValue.Interface().(HTTPClient)
	case option.Client != nil:
		r.httpClient = option.Client
	default:
		r.tlsConfDir = gotOption(tlsConfDirType, "tlsConfDir", option.TLSConfDir, f, numIn, args)
		r.tlsConfFiles = gotOption(tlsConfFilesType, "tlsConfFiles", option.TLSConfFiles, f, numIn, args)
		r.keepalive = gotOption(keepaliveType, "keepalive", option.Keepalive, f, numIn, args)
		r.httpClient = &http.Client{Transport: r.transport()}
	}

	return r, nil
}

func makeFunc(option *Option, f StructField, numIn, numOut int) generalFn {
	return func(args []reflect.Value) ([]reflect.Value, error) {
		runner, err := newRunner(option, f, numIn, args)
		if err != nil {
			return nil, err
		}

		req, rsp, err := runner.httpClientDo()
		if err != nil {
			return nil, err
		}

		defer rsp.Body.Close()

		dlValue := findArgs(f, numIn, args, dlFilePtrType)
		if dlValue.IsValid() {
			if err := processDl(dlValue, rsp); err != nil {
				return nil, err
			}
		}

		runner.dumpRsp(req, rsp, dlValue)

		if numOut == 0 {
			return []reflect.Value{}, nil
		}

		if dlValue.IsValid() {
			return nil, fmt.Errorf("download file has alread read all the response body")
		}

		return processOut(f, rsp, numOut)
	}
}

func (r *runner) httpClientDo() (*http.Request, *http.Response, error) {
	body, contentType, isFileUpload, err := parseBodyContentType(r.inputs)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequest(r.method, r.addr, body)
	if err != nil {
		return nil, nil, err
	}

	if contentType != "" {
		req.Header.Set(gonet.ContentType, contentType)
	}

	r.dumpReq(req, isFileUpload)

	rsp, err := r.httpClient.Do(req)

	return req, rsp, err
}

func parseURL(args []reflect.Value, option *Option, f StructField, numIn int) (string, error) {
	u := gotOption(urlType, "u", option.URL, f, numIn, args)
	if u == "" && option.urlField.IsValid() {
		u = option.urlField.Convert(stringType).Interface().(string)
	}

	if u == "" {
		return "", fmt.Errorf("URL not specified")
	}

	return u, nil
}

func (r *runner) dumpRsp(req *http.Request, rsp *http.Response, dlValue reflect.Value) {
	logrus.Debugf("[%s %s] StatusCode[%d]", req.Method, req.URL.String(), rsp.StatusCode)

	if !strings.Contains(r.dumpOption, "rsp") {
		return
	}

	d, err := httputil.DumpResponse(rsp, !dlValue.IsValid())
	if err != nil {
		r.option.Logger.LogError(err)
		return
	}

	if l, ok := r.option.Logger.(DumpResponseLogger); ok {
		l.Dump(d)
		return
	}

	logrus.Infof("Response:\n%s\n", d)
}

func (r *runner) dumpReq(req *http.Request, isFileUpload bool) {
	if !strings.Contains(r.dumpOption, "req") {
		return
	}

	d, err := httputil.DumpRequest(req, !isFileUpload)
	if err != nil {
		r.option.Logger.LogError(err)
		return
	}

	if l, ok := r.option.Logger.(DumpRequestLogger); ok {
		l.Dump(d)
		return
	}

	logrus.Infof("Request:\n%s\n", d)
}

func parseBodyContentType(inputs []reflect.Value) (io.Reader, string, bool, error) {
	if len(inputs) > 0 {
		return createBody(inputs)
	}

	return nil, "", false, nil
}

func processDl(dlValue reflect.Value, res *http.Response) error {
	dl := dlValue.Interface().(*DownloadFile)
	if _, err := io.Copy(dl.Writer, res.Body); err != nil {
		return err
	}

	dl.Filename = decodeDlFilename(res)

	return nil
}

func decodeDlFilename(res *http.Response) string {
	// decode w.Header().Set("Content-Disposition", "attachment; filename=WHATEVER_YOU_WANT")
	if cd := res.Header.Get("Content-Disposition"); cd != "" {
		if _, params, err := mime.ParseMediaType(cd); err == nil {
			return params["filename"]
		}
	}

	return ""
}

func createBody(inputs []reflect.Value) (io.Reader, string, bool, error) {
	fileValue := findInputByType(inputs, fileType)

	if !fileValue.IsValid() {
		j, _ := json.Marshal(inputs[0].Interface())

		return bytes.NewReader(j), HeadJSON, false, nil
	}

	r, contentType, err := prepareFile(inputs, fileValue)

	return r, contentType, true, err
}

func prepareFile(inputs []reflect.Value, fileValue reflect.Value) (io.Reader, string, error) {
	file := fileValue.Interface().(UploadFile)

	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)

	formFile, err := writer.CreateFormFile(file.FilenameKey, file.Filename)
	if err != nil {
		return nil, "", fmt.Errorf("create form file failed: %w", err)
	}

	_, err = io.Copy(formFile, file.Reader)

	if err != nil {
		return nil, "", fmt.Errorf("write to form file failed: %w", err)
	}

	if params := findInputByType(inputs, paramsType); params.IsValid() {
		for k, v := range params.Interface().(map[string]string) {
			_ = writer.WriteField(k, v)
		}
	}

	contentType := writer.FormDataContentType()

	_ = writer.Close() // 发送之前必须调用Close()以写入结尾行

	return buf, contentType, nil
}

func findInputByType(inputs []reflect.Value, typ reflect.Type) reflect.Value {
	for _, input := range inputs {
		if input.Type() == typ {
			return input
		}
	}

	return emptyValue
}

// DrainBody reads all of b to memory and then returns two equivalent
// ReadClosers yielding the same bytes.
//
// It returns an error if the initial slurp of all bytes fails. It does not attempt
// to make the returned ReadClosers have identical error-matching behavior.
func DrainBody(b io.ReadCloser) (r1 []byte, r2 io.ReadCloser, err error) {
	if b == nil || b == http.NoBody {
		// No copying needed. Preserve the magic sentinel meaning of NoBody.
		return nil, http.NoBody, nil
	}

	var buf bytes.Buffer

	if _, err = buf.ReadFrom(b); err != nil {
		return nil, b, err
	}

	if err = b.Close(); err != nil {
		return nil, b, err
	}

	return buf.Bytes(), ioutil.NopCloser(bytes.NewReader(buf.Bytes())), nil
}

func processOut(f StructField, res *http.Response, outNum int) ([]reflect.Value, error) {
	var (
		bodyBytes []byte
		err       error
	)

	outs := make([]reflect.Value, 0, outNum)

	for i := 0; i < outNum; i++ {
		outType := f.Type.Out(i)
		if outType == reflect.TypeOf(res) {
			outs = append(outs, reflect.ValueOf(res))
			continue
		}

		if bodyBytes == nil {
			if bodyBytes, res.Body, err = DrainBody(res.Body); err != nil {
				return nil, err
			}
		}

		if res.StatusCode < 200 || res.StatusCode >= 300 {
			return nil, fmt.Errorf("unexpected status code %d with  response %s",
				res.StatusCode, string(gonet.ReadBytes(res.Body)))
		}

		switch outType.Kind() {
		case reflect.Struct:
			outVPtr := reflect.New(outType)
			if err := json.Unmarshal(bodyBytes, outVPtr.Interface()); err != nil {
				return nil, err
			}

			outs = append(outs, outVPtr.Elem())
		case reflect.String:
			outs = append(outs, reflect.ValueOf(string(bodyBytes)))
		default:
			any, err := gor.CastAny(string(bodyBytes), outType)
			if err != nil {
				return nil, err
			}

			outs = append(outs, any)
		}
	}

	return outs, nil
}

type transportKey struct {
	timeout                  time.Duration
	tlsConfDir, tlsConfFiles string
}

// nolint gochecknoglobals
var (
	transportMap     = make(map[transportKey]*http.Transport)
	transportMapLock sync.RWMutex
)

func (r *runner) transport() *http.Transport {
	if !Keepalive(r.keepalive).IsKeepAlive() {
		return &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: r.timeout, KeepAlive: r.timeout}).DialContext,
			IdleConnTimeout:       r.timeout,
			TLSHandshakeTimeout:   r.timeout,
			ExpectContinueTimeout: r.timeout,
			DisableKeepAlives:     true,
			MaxIdleConnsPerHost:   -1,

			TLSClientConfig: parseTLSConfig(r.tlsConfDir, r.tlsConfFiles),
		}
	}

	key := transportKey{
		timeout:      r.timeout,
		tlsConfDir:   r.tlsConfDir,
		tlsConfFiles: r.tlsConfFiles,
	}

	transportMapLock.RLock()
	t, ok := transportMap[key]
	transportMapLock.RUnlock()

	if ok {
		return t
	}

	transportMapLock.Lock()
	defer transportMapLock.Unlock()

	t = &http.Transport{
		Proxy:        http.ProxyFromEnvironment,
		DialContext:  (&net.Dialer{Timeout: r.timeout, KeepAlive: r.timeout}).DialContext,
		MaxIdleConns: 100, // nolint gomnd

		IdleConnTimeout:       r.timeout,
		TLSHandshakeTimeout:   r.timeout,
		ExpectContinueTimeout: r.timeout,
		MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,

		TLSClientConfig: parseTLSConfig(r.tlsConfDir, r.tlsConfFiles),
	}

	transportMap[key] = t

	return t
}

func parseTLSConfig(tlsConfDir, tlsConfFiles string) *tls.Config {
	c := strings.SplitN(tlsConfFiles, ",", 3) // clientKeyFile,clientCertFile,serverRootCA
	if len(c) != 3 {                          // nolint gomnd
		tc := &tls.Config{} // nolint gosec
		tlsconf.SkipHostnameVerification(tc)

		return tc
	}

	if tlsConfDir != "" {
		for i, p := range c {
			c[i] = filepath.Join(tlsConfDir, p)
		}
	}

	return tlsconf.CreateClient(c[0], c[1], c[2])
}

func gotInputs(f StructField, numIn int, args []reflect.Value) []reflect.Value {
	var inputs []reflect.Value

	for i := 0; i < numIn; i++ {
		if inputType(f.Type.In(i)) {
			inputs = append(inputs, args[i])
		}
	}

	return inputs
}

func findArgsImpl(f StructField, numIn int, args []reflect.Value, typ reflect.Type) reflect.Value {
	for i := 0; i < numIn; i++ {
		if gor.ImplType(f.Type.In(i), typ) {
			return args[i]
		}
	}

	return emptyValue
}

func findArgs(f StructField, numIn int, args []reflect.Value, typ reflect.Type) reflect.Value {
	for i := 0; i < numIn; i++ {
		if f.Type.In(i) == typ {
			return args[i]
		}
	}

	return emptyValue
}

func gotOption(typ reflect.Type, tag, defaultValue string, f StructField, numIn int, args []reflect.Value) string {
	if typ != nil {
		for i := 0; i < numIn; i++ {
			if f.Type.In(i) == typ {
				return args[i].Convert(stringType).Interface().(string)
			}
		}
	}

	if v := f.Tag.Get(tag); v != "" {
		return v
	}

	return defaultValue
}

func makeOption(structValue *StructValue, manv reflect.Value, optionFns []OptionFn) *Option {
	o := &Option{}

	for _, fn := range optionFns {
		fn(o)
	}

	if o.URL == "" {
		o.urlField, o.URL = findOption(urlType, "url", "", structValue, manv)
	}

	if o.Method == "" {
		_, o.Method = findOption(methodType, "method", "GET", structValue, manv)
	}

	if o.Keepalive == "" {
		_, o.Keepalive = findOption(keepaliveType, "keepalive", "true", structValue, manv)
	}

	if o.Timeout == "" {
		_, o.Timeout = findOption(timeoutType, "timeout", "90s", structValue, manv)
	}

	if o.TLSConfDir == "" {
		_, o.TLSConfDir = findOption(tlsConfDirType, "tlsConfDir", "", structValue, manv)
	}

	if o.TLSConfFiles == "" {
		_, o.TLSConfFiles = findOption(tlsConfFilesType, "tlsConfFiles", "", structValue, manv)
	}

	createErrorSetter(o)
	createLogger(manv, o)

	return o
}

func findOption(typ reflect.Type, tag, defValue string, sv *StructValue, manv reflect.Value) (reflect.Value, string) {
	for i := 0; i < manv.NumField(); i++ {
		if f := manv.Field(i); f.Type() == typ {
			return f, f.Convert(stringType).Interface().(string)
		}
	}

	for i := 0; i < sv.NumField; i++ {
		if ft := sv.FieldTypes[i]; ft.Type == tType {
			if v := ft.Tag.Get(tag); v != "" {
				return emptyValue, v
			}
		}
	}

	return emptyValue, defValue
}

// StructField represents the information of a struct's field
type StructField struct {
	Parent      *StructValue
	Field       reflect.Value
	Index       int
	StructField reflect.StructField
	Type        reflect.Type
	Name        string
	Tag         reflect.StructTag
	Kind        reflect.Kind
	PkgPath     string
}

// StructValue represents the
type StructValue struct {
	StructSelf reflect.Value
	NumField   int
	FieldTypes []reflect.StructField
}

// MakeStructValue makes a StructValue by a struct's value.
func MakeStructValue(structSelf reflect.Value) *StructValue {
	sv := &StructValue{StructSelf: structSelf, NumField: structSelf.NumField()}

	sv.FieldTypes = make([]reflect.StructField, sv.NumField)
	for i := 0; i < sv.NumField; i++ {
		sv.FieldTypes[i] = sv.StructSelf.Type().Field(i)
	}

	return sv
}

// FieldByIndex return the StructField at index
func (s *StructValue) FieldByIndex(index int) StructField {
	fieldType := s.FieldTypes[index]
	field := s.StructSelf.Field(index)

	return StructField{
		Parent:      s,
		Field:       field,
		Index:       index,
		StructField: fieldType,
		Type:        fieldType.Type,
		Name:        fieldType.Name,
		Tag:         fieldType.Tag,
		Kind:        field.Kind(),
		PkgPath:     fieldType.PkgPath,
	}
}

// HTTPClient defines the interface for HTTP client.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// nolint gochecknoglobals
var (
	emptyValue reflect.Value

	httpClientType   = reflect.TypeOf((*HTTPClient)(nil)).Elem()
	dlFilePtrType    = reflect.TypeOf((*DownloadFile)(nil))
	paramsType       = reflect.TypeOf((*map[string]string)(nil)).Elem()
	fileType         = reflect.TypeOf((*UploadFile)(nil)).Elem()
	keepaliveType    = reflect.TypeOf((*Keepalive)(nil)).Elem()
	timeoutType      = reflect.TypeOf((*Timeout)(nil)).Elem()
	tType            = reflect.TypeOf((*T)(nil)).Elem()
	urlType          = reflect.TypeOf((*URL)(nil)).Elem()
	methodType       = reflect.TypeOf((*Method)(nil)).Elem()
	stringType       = reflect.TypeOf((*string)(nil)).Elem()
	manLoggerType    = reflect.TypeOf((*Logger)(nil)).Elem()
	tlsConfFilesType = reflect.TypeOf((*TLSConfFiles)(nil)).Elem()
	tlsConfDirType   = reflect.TypeOf((*TLSConfDir)(nil)).Elem()
)

func inputType(t reflect.Type) bool {
	switch t {
	case methodType, urlType, timeoutType, keepaliveType, dlFilePtrType,
		tlsConfFilesType, tlsConfDirType, httpClientType, tType:
		return false
	}

	return true
}

// IsKeepAlive tells the keepalive option is enabled or not.
func (k Keepalive) IsKeepAlive() bool {
	switch strings.ToLower(string(k)) {
	case "false", "no", "off", "0":
		return false
	default: // "true", "yes", "on", "1", etc.
		return true
	}
}

func createErrorSetter(option *Option) {
	option.ErrSetter = func(err error) {
		if err == nil {
			return
		}

		logrus.Warnf("error occurred %v", err)
	}
}

func createLogger(v reflect.Value, option *Option) {
	if fv := findTypedField(v, manLoggerType); fv.IsValid() {
		option.Logger = fv.Interface().(Logger)
		return
	}

	option.Logger = &LoggerNoop{}
}

// LoggerNoop implements the interface for dao logging with NOOP.
type LoggerNoop struct{}

// LogError logs the error
func (d *LoggerNoop) LogError(err error) { /*NOOP*/ }

func findTypedField(v reflect.Value, t reflect.Type) reflect.Value {
	for i := 0; i < v.NumField(); i++ {
		f := v.Type().Field(i)

		if f.PkgPath != "" /* not exportable? */ {
			continue
		}

		fv := v.Field(i)
		if gor.ImplType(f.Type, t) && !fv.IsNil() {
			return fv
		}
	}

	return reflect.Value{}
}

// Logger is the interface for http logging.
type Logger interface {
	// LogError logs the error
	LogError(err error)
}

// DumpRequestLogger is the interface for http dump.
type DumpRequestLogger interface {
	// Dump logs the dmp
	Dump(dump []byte)
}

// DumpResponseLogger is the interface for http dump.
type DumpResponseLogger interface {
	// Dump logs the dmp
	Dump(dump []byte)
}

// QueryURL composes the GET url with query arguments
func QueryURL(baseURL string, kvs ...string) URL {
	u, _ := url.Parse(baseURL)
	q, _ := url.ParseQuery(u.RawQuery)

	for i := 0; i < len(kvs); i += 2 {
		k, v := kvs[i], ""

		if i+1 < len(kvs) {
			v = kvs[i+1]
		}

		q.Add(k, v)
	}

	u.RawQuery = q.Encode()

	return URL(u.String())
}
