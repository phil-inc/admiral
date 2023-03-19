package logstream

import (
	"bufio"
	"crypto/rand"
	"io"
	"testing"
	"time"

	"github.com/phil-inc/admiral/pkg/backend"
	"github.com/phil-inc/admiral/pkg/state"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

var mocked_pod = &v1.Pod{}
var mocked_container = v1.Container{}

func Test_Logstream(t *testing.T) {
	st := state.New("test-cluster")
	rawLogCh := make(chan backend.RawLog)

	errCh := make(chan error)
	st.SetErrChannel(errCh)

	l := New().RawLogChannel(rawLogCh).State(st).Container(mocked_container).Pod(mocked_pod).Build()
	assert.NotNil(t, l)

	results := []backend.RawLog{}

	go func(){
		for raw := range rawLogCh {
			results = append(results, raw)
		}
	}()

	// create a pipe simulating an io.ReadCloser
	reader, writer := io.Pipe()

	// create a buffer that will write to the pipe
	bufferedWriter := bufio.NewWriter(writer)

	// wire the buffers to the logstream
	l.stream = io.NopCloser(reader)
	l.reader = bufio.NewReader(l.stream)

	go l.Read()

	// write to the pipe
	for i := 0; i < 10; i++ {
		// make some random data
		b := make([]byte, 16)
		rand.Read(b)
		b[len(b)-1] = byte('\n')

		// write it into the buffer
		bufferedWriter.Write(b)
		bufferedWriter.Flush()
	}

	// give the test a chance to catch up
	time.Sleep(5 * time.Second)

	writer.Close()
	<-errCh

	assert.Len(t, results, 10)
}

