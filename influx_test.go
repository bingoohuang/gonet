package gonet

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInfluxWrite(t *testing.T) {
	ti := time.Now()
	line, err := LineProtocol("weather",
		map[string]string{"location": "us-midwest"},
		map[string]interface{}{"temperature": 82}, ti) // nolint gomnd

	if err != nil {
		t.Fatal(err)
	}

	a := assert.New(t)
	a.Equal(fmt.Sprintf("%s %d", "weather,location=us-midwest temperature=82", ti.UnixNano()), line)
}

//if err := InfluxWrite("http://beta.isignet.cn:10014/write?db=metrics", line); err != nil {
//	t.Fatal(err)
//}
