package loki

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func Test_Build(t *testing.T) {
	cli := &http.Client{}
	ch := make(chan backend.RawLog)

	l := New().Client(cli).LogChannel(ch).Url("loki.com").Build()

	assert.NotNil(t, l)
	assert.Equal(t, "loki.com/loki/api/v1/push", l.url)
	assert.NotNil(t, l.client)
	assert.Equal(t, ch, l.logChannel)
}

func Test_rawLogToDTO(t *testing.T) {
	l := New().Build()
	expected := &lokiDTO{
		Streams: []streams{
			{
				Stream: map[string]string{"hello": "world"},
				Values: [][]string{{"hello world"}},
			},
		},
	}

	r := backend.RawLog{
		Log: "hello world",
		Metadata: map[string]string{
			"hello": "world",
		},
	}

	actual := l.rawLogToDTO(r)

	assert.Equal(t, expected.Streams[0].Stream, actual.Streams[0].Stream)
	assert.Equal(t, expected.Streams[0].Values[0][0], actual.Streams[0].Values[0][1])

}

func Test_Concurrency(t *testing.T) {
	received := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received++
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		b, err := ioutil.ReadAll(r.Body)

		assert.Nil(t, err)
		assert.Contains(t, string(b), "some log")
		assert.Contains(t, string(b), "some metadata")
		assert.Contains(t, string(b), "other metadata")
	}))

	ch := make(chan backend.RawLog)
	cli := &http.Client{}
	errCh := make(chan error)
	l := New().ErrChannel(errCh).LogChannel(ch).Url(server.URL).Client(cli).Build()

	go utils.HandleErrorStream(errCh)
	defer close(errCh)
	go l.Stream()

	count := 50000
	var wg sync.WaitGroup
	wg.Add(count)

	for i := 0; i < count; i++ {
		go func() {
			defer wg.Done()

			r := backend.RawLog{
				Log: "some log",
				Metadata: map[string]string{
					"hello": "some metadata",
					"world": "other metadata",
				},
			}
			l.logChannel <- r
		}()
	}

	wg.Wait()

	for {
		if received == count {
			break
		}
	}

	l.Close()
}

func Test_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		b, err := ioutil.ReadAll(r.Body)

		assert.Nil(t, err)
		assert.Contains(t, string(b), "some log")
		assert.Contains(t, string(b), "some metadata")
		assert.Contains(t, string(b), "other metadata")
	}))

	cli := &http.Client{}
	ch := make(chan backend.RawLog)
	errCh := make(chan error)

	l := New().ErrChannel(errCh).Url(server.URL).Client(cli).LogChannel(ch).Build()

	go l.Stream()

	raw := backend.RawLog{
		Log: "some log",
		Metadata: map[string]string{
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
	ch := make(chan backend.RawLog)
	errCh := make(chan error)

	l := New().ErrChannel(errCh).Url(server.URL).Client(cli).LogChannel(ch).Build()

	go utils.HandleErrorStream(errCh)
	defer close(errCh)
	go l.Stream()

	raw := backend.RawLog{
		Log: "some log",
		Metadata: map[string]string{
			"hello": "some metadata",
			"world": "other metadata",
		},
	}

	l.logChannel <- raw

	l.Close()
}
