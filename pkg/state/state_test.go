package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_SharedMutable(t *testing.T) {
	s := New()
	assert.NotNil(t, s)

	time := s.InitTimestamp()
	assert.NotNil(t, time)
	
	req := Request{
		Key: "hello",
		Value: "world",
	}

	s.objectChannel <- req
	assert.Equal(t, "world", s.objects["hello"])
}
