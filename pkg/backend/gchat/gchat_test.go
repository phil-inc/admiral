package gchat

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
	ch := make(chan string)

	g := New().Client(cli).TextChannel(ch).Url("gchat.com").Build()

	assert.NotNil(t, g)
	assert.Equal(t, "gchat.com", g.url)
	assert.NotNil(t, g.client)
	assert.Equal(t, ch, g.textChannel)
}

func Test_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

		b, err := ioutil.ReadAll(r.Body)
		assert.Nil(t, err)
		assert.Contains(t, "some event", b)
	}))

	cli := &http.Client{}
	ch := make(chan string)
	errCh := make(chan error)

	g := New().Client(cli).Url(server.URL).TextChannel(ch).ErrChannel(errCh).Build()

	go g.Stream()

	msg := "some event"

	g.textChannel <- msg

	g.Close()
}

func Test_StreamErr(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1")
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte(`<h1>hello world</h1>`))
	}))

	cli := &http.Client{}
	ch := make(chan string)
	errCh := make(chan error)

	g := New().ErrChannel(errCh).Url(server.URL).Client(cli).TextChannel(ch).Build()

	go utils.HandleErrorStream(errCh)
	defer close(errCh)
	go g.Stream()

	msg := "my event"

	g.textChannel <- msg

	g.Close()
}
