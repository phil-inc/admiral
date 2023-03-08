package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_SharedMutable(t *testing.T) {
	s := New("hello")
	assert.NotNil(t, s)
	assert.Equal(t, "hello", s.Cluster())

	timestamp := s.InitTimestamp()
	assert.NotNil(t, timestamp)

	s.Set("hello", "world")
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, "world", s.Get("hello"))

	s.Delete("hello")
	time.Sleep(1 * time.Millisecond)
	assert.Equal(t, "", s.Get("hello"))
}
