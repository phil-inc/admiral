package loki

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Build(t *testing.T) {
	cli := &http.Client{}
	ch := make(chan string)

	l := New().Client(cli).LogChannel(ch).Url("loki.com").Build()

	assert.NotNil(t, l)
	assert.Equal(t, "loki.com/loki/api/v1/push", l.url)
	assert.NotNil(t, l.client)
	assert.Equal(t, ch, l.logChannel)
}

func Test_Stream(t *testing.T) {
	l := New().Build()
	l.Stream()
}
