package loki

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Build(t *testing.T) {
	cli := &http.Client{}
	ch := make(chan RawLog)

	l := New().Client(cli).LogChannel(ch).Url("loki.com").Build()

	assert.NotNil(t, l)
	assert.Equal(t, "loki.com/loki/api/v1/push", l.url)
	assert.NotNil(t, l.client)
	assert.Equal(t, ch, l.logChannel)
}

func Test_rawLogToDTO(t *testing.T) {
	expected := &lokiDTO{
		streams: []streams{
			{
				stream: map[string]string{"hello":"world",},
				values: [][]string{{"hello world"}},
			},
		},
	}
	
	r := RawLog{
		log: "hello world",
		metadata: map[string]string{
			"hello": "world",
		},
	}

	actual := rawLogToDTO(r)

	assert.Equal(t, expected, actual)

}

func Test_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		b, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Contains(t, b, "some log")
		assert.Contains(t, b, "some metadata")
		assert.Contains(t, b, "other metadata")
	}))

	cli := &http.Client{}
	ch := make(chan RawLog)
	errCh := make(chan error)

	l := New().ErrChannel(errCh).Url(server.URL).Client(cli).LogChannel(ch).Build()

	go l.Stream()

	raw := RawLog{
		log: "some log",
		metadata: map[string]string{
			"hello": "some metadata",
			"world": "other metadata",
		},
	}

	l.logChannel <- raw

	l.Close()
}

func Test_sendRequestRespErr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(`<h1>hello world</h1>`))
	}))

	cli := &http.Client{}
	ch := make(chan RawLog)
	errCh := make(chan error)

	l := New().ErrChannel(errCh).Url(server.URL).Client(cli).LogChannel(ch).Build()

	go utils.HandleErrorStream(errCh)
	defer close(errCh)
	go l.Stream()

	raw := RawLog{
		log: "some log",
		metadata: map[string]string{
			"hello": "some metadata",
			"world": "other metadata",
		},
	}

	l.logChannel <- raw

	l.Close()
}
