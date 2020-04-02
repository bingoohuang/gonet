package influx

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/bingoohuang/gonet"
)

// Query Query execute influxQl (refer to https://docs.influxdata.com/influxdb/v1.7/query_language)
// influxDBAddr  InfluxDB的连接地址， 例如http://localhost:8086, 注意：1. 右边没有/ 2. 右边不带其它path，例如/query等。
func Query(influxDBAddr, influxQl string) (string, error) {
	req, err := gonet.Get(influxDBAddr + `/query`)
	if err != nil {
		return "", err
	}

	req.Param("q", influxQl)

	return req.String()
}

// Write 写入打点值
// refer https://github.com/DCSO/fluxline/blob/master/encoder.go
func Write(influxDBWriteAddr, line string) (*http.Response, string, error) {
	req, err := gonet.Post(influxDBWriteAddr)
	if err != nil {
		return nil, "", err
	}

	req.Body([]byte(line))

	rsp, err := req.SendOut()
	if err != nil {
		return rsp, "", err
	}

	rspBody, err := req.ReadResponseBody(rsp)

	return rsp, string(rspBody), err
}

// LineProtocol format inputs to line protocol
// https://docs.influxdata.com/influxdb/v1.7/write_protocols/line_protocol_tutorial/
func LineProtocol(name string, tags map[string]string, fields map[string]interface{}, t time.Time) (string, error) {
	if len(fields) == 0 {
		return "", errors.New("fields are empty")
	}

	tagKeys := make([]string, 0, len(tags))

	for k, v := range tags {
		if k != "" && v != "" {
			tagKeys = append(tagKeys, k)
		}
	}

	sort.Strings(tagKeys)

	tagstr := ""
	for _, k := range tagKeys {
		tagstr += fmt.Sprintf(",%s=%s", escapeSpecialChars(k), escapeSpecialChars(tags[k]))
	}

	fieldKeys := make([]string, 0, len(fields))

	for k, v := range fields {
		if k != "" && v != nil {
			fieldKeys = append(fieldKeys, k)
		}
	}

	sort.Strings(fieldKeys)

	out := ""

	for _, k := range fieldKeys {
		repr, err := toInfluxRepr(fields[k])
		if err != nil {
			return "", err
		}

		out += fmt.Sprintf(",%s=%s", escapeSpecialChars(k), repr)
	}

	if out != "" {
		out = out[1:]
	}

	// construct line protocol string
	return fmt.Sprintf("%s%s %s %d", name, tagstr, out, uint64(t.UnixNano())), nil
}

func escapeSpecialChars(in string) string {
	str := strings.Replace(in, ",", `\,`, -1)
	str = strings.Replace(str, "=", `\=`, -1)
	str = strings.Replace(str, " ", `\ `, -1)

	return str
}

// toInfluxRepr 将val转换为Influx表示形式
func toInfluxRepr(val interface{}) (string, error) {
	switch v := val.(type) {
	case string:
		return stringToInfluxRepr(v)
	case []byte:
		return stringToInfluxRepr(string(v))
	case int32, int64, int16, int8, int, uint32, uint64, uint16, uint8, uint:
		return fmt.Sprintf("%d", v), nil
	case float64, float32:
		return fmt.Sprintf("%g", v), nil
	case bool:
		return fmt.Sprintf("%t", v), nil
	case time.Time:
		return fmt.Sprintf("%d", uint64(v.UnixNano())), nil
	default:
		return "", fmt.Errorf("%+v: unsupported type for Influx Line Protocol", val)
	}
}

func stringToInfluxRepr(v string) (string, error) {
	if len(v) > 64000 { // nolint gomnd
		return "", fmt.Errorf("string too long (%d characters, max. 64K)", len(v))
	}

	return fmt.Sprintf("%q", v), nil
}
