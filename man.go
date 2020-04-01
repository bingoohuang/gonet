package gonet

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/bingoohuang/gor"
)

// T is the a special interface for additional tag for the man1.
type T interface{ t() }

// URL is the URL address for the http requests.
type URL string

// Method is the HTTP Method for the http requests.
type Method string

// Timeout is the timeout setting for the http requests.
type Timeout string

// KeepAlive is the keep-alive flag (true(on,yes,1) /false(no, off,0), default true.
type KeepAlive string

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

// TLS specifies the TLS configuration for the client.
type TLS string

// MakeTLS makes a TLS configuration for the client
func MakeTLS(clientKeyFile, clientCertFile, serverRootCA string) TLS {
	return TLS(clientKeyFile + "," + clientCertFile + "," + serverRootCA)
}

// ManOption is the options for Man.
type ManOption struct {
	// URL ...
	URL string

	urlField reflect.Value

	// Method ...
	Method string
	// KeepAlive ...
	KeepAlive string
	// Timeout ...
	Timeout string
	// TLS 	clientKeyFile,clientCertFile,serverRootCA(required=false)
	TLS string

	ErrSetter func(err error)
	Logger    ManLogger
}

// ManOptionFn is the func prototype for ManOption.
type ManOptionFn func(*ManOption)

// NewMan makes a new Man for http requests.
func NewMan(man interface{}, optionFns ...ManOptionFn) error {
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

func createFn(option *ManOption, f StructField) error {
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
			values = append(values, reflect.ValueOf(err))
		}

		return values
	}))

	return nil
}

type generalFn func(args []reflect.Value) ([]reflect.Value, error)

func makeFunc(option *ManOption, f StructField, numIn int, numOut int) generalFn {
	return func(args []reflect.Value) ([]reflect.Value, error) {
		method := gotOption(methodType, "method", option.Method, f, numIn, args)
		tls := gotOption(tlsType, "tls", option.TLS, f, numIn, args)
		dumpOption := gotOption(nil, "dump", option.Method, f, numIn, args)
		inputs := gotInputs(f, numIn, args)

		//keepAlive := gotKeepAlive(option, f, numIn, args)
		timeout := gotOption(timeoutType, "timeout", option.Timeout, f, numIn, args)
		timeoutDuration, err := time.ParseDuration(timeout)

		if err != nil {
			return nil, fmt.Errorf("failed to parse timeout %s error: %v", timeout, err)
		}

		u, err := parseURL(args, option, f, numIn)
		if err != nil {
			return nil, err
		}

		rsp, err := do(tls, inputs, method, u, dumpOption, option, timeoutDuration)
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

		dumpRsp(dumpOption, rsp, dlValue, option)

		if numOut == 0 {
			return []reflect.Value{}, nil
		}

		if dlValue.IsValid() {
			return nil, fmt.Errorf("download file has alread read all the response body")
		}

		return processOut(f, rsp)
	}
}

func do(tls string, inputs []reflect.Value, method, url, dumpOption string,
	option *ManOption, timeoutDuration time.Duration) (*http.Response, error) {
	body, contentType, isFileUpload, err := parseBodyContentType(inputs)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if contentType != "" {
		req.Header.Set(ContentType, contentType)
	}

	dumpReq(dumpOption, req, isFileUpload, option)

	c := &http.Client{Transport: transport(timeoutDuration, tls)}

	return c.Do(req)
}

func parseURL(args []reflect.Value, option *ManOption, f StructField, numIn int) (string, error) {
	url := gotOption(urlType, "url", option.URL, f, numIn, args)
	if url == "" && option.urlField.IsValid() {
		url = option.urlField.Convert(stringType).Interface().(string)
	}

	if url == "" {
		return "", fmt.Errorf("URL not specified")
	}

	return url, nil
}

func dumpRsp(dumpOption string, rsp *http.Response, dlValue reflect.Value, option *ManOption) {
	if !strings.Contains(dumpOption, "rsp") {
		return
	}

	d, err := httputil.DumpResponse(rsp, !dlValue.IsValid())
	if err != nil {
		option.Logger.LogError(err)
		return
	}

	if l, ok := option.Logger.(DumpResponseLogger); ok {
		l.Dump(d)
		return
	}

	logrus.Infof("Response:\n%s\n", d)
}

func dumpReq(dumpOption string, req *http.Request, isFileUpload bool, option *ManOption) {
	if !strings.Contains(dumpOption, "req") {
		return
	}

	d, err := httputil.DumpRequest(req, !isFileUpload)
	if err != nil {
		option.Logger.LogError(err)
		return
	}

	if l, ok := option.Logger.(DumpRequestLogger); ok {
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

func processOut(f StructField, res *http.Response) ([]reflect.Value, error) {
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status code %d with  response %s",
			res.StatusCode, string(ReadBytes(res.Body)))
	}

	outType := f.Type.Out(0)
	outVPtr := reflect.New(outType)
	bodyBytes := ReadBytes(res.Body)

	if err := json.Unmarshal(bodyBytes, outVPtr.Interface()); err != nil {
		return nil, err
	}

	return []reflect.Value{outVPtr.Elem()}, nil
}

func transport(timeout time.Duration, tlsConfig string) *http.Transport {
	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   timeout,
			KeepAlive: timeout,
		}).DialContext,
		//MaxIdleConns:          100,
		IdleConnTimeout:       timeout,
		TLSHandshakeTimeout:   timeout,
		ExpectContinueTimeout: timeout,
		//MaxIdleConnsPerHost:   runtime.GOMAXPROCS(0) + 1,

		DisableKeepAlives:   true,
		MaxIdleConnsPerHost: -1,

		TLSClientConfig: parseTLSConfig(tlsConfig),
	}
}

func parseTLSConfig(tlsConfig string) *tls.Config {
	c := strings.SplitN(tlsConfig, ",", 3) // clientKeyFile,clientCertFile,serverRootCA
	if len(c) != 3 {                       // nolint gomnd
		return &tls.Config{InsecureSkipVerify: true} // nolint gosec
	}

	return TLSConfigCreateClient(c[1], c[2], c[3])
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

func makeOption(structValue *StructValue, manv reflect.Value, optionFns []ManOptionFn) *ManOption {
	o := &ManOption{}

	for _, fn := range optionFns {
		fn(o)
	}

	if o.URL == "" {
		o.urlField, o.URL = findOption(urlType, "url", "", structValue, manv)
	}

	if o.Method == "" {
		_, o.Method = findOption(methodType, "method", "GET", structValue, manv)
	}

	if o.KeepAlive == "" {
		_, o.KeepAlive = findOption(keepAliveType, "keepalive", "true", structValue, manv)
	}

	if o.Timeout == "" {
		_, o.Timeout = findOption(keepAliveType, "timeout", "90s", structValue, manv)
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

// nolint gochecknoglobals
var (
	emptyValue reflect.Value

	dlFilePtrType = reflect.TypeOf((*DownloadFile)(nil))
	paramsType    = reflect.TypeOf((*map[string]string)(nil)).Elem()
	fileType      = reflect.TypeOf((*UploadFile)(nil)).Elem()
	keepAliveType = reflect.TypeOf((*KeepAlive)(nil)).Elem()
	timeoutType   = reflect.TypeOf((*Timeout)(nil)).Elem()
	tType         = reflect.TypeOf((*T)(nil)).Elem()
	urlType       = reflect.TypeOf((*URL)(nil)).Elem()
	methodType    = reflect.TypeOf((*Method)(nil)).Elem()
	stringType    = reflect.TypeOf((*string)(nil)).Elem()
	manLoggerType = reflect.TypeOf((*ManLogger)(nil)).Elem()
	tlsType       = reflect.TypeOf((*TLS)(nil)).Elem()
)

func inputType(t reflect.Type) bool {
	switch t {
	case methodType, urlType, timeoutType, keepAliveType, dlFilePtrType, tlsType:
		return false
	}

	return true
}

// IsKeepAlive tells the keepalive option is enabled or not.
func (k KeepAlive) IsKeepAlive() bool {
	switch strings.ToLower(string(k)) {
	case "false", "no", "off", "0":
		return false
	case "true", "yes", "on", "1":
		return true
	default:
		return true
	}
}

func createErrorSetter(option *ManOption) {
	option.ErrSetter = func(err error) {
		if err == nil {
			return
		}

		logrus.Warnf("error occurred %v", err)
	}
}

func createLogger(v reflect.Value, option *ManOption) {
	if fv := findTypedField(v, manLoggerType); fv.IsValid() {
		option.Logger = fv.Interface().(ManLogger)
		return
	}

	option.Logger = &ManLoggerNoop{}
}

// ManLoggerNoop implements the interface for dao logging with NOOP.
type ManLoggerNoop struct{}

// LogError logs the error
func (d *ManLoggerNoop) LogError(err error) { /*NOOP*/ }

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

// ManLogger is the interface for http logging.
type ManLogger interface {
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

// QueryURL comoses the GET url with query arguments
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