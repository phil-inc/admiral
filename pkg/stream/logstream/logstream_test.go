package logstream

import (
	"bufio"
	"bytes"
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

	go func(){
		for raw := range rawLogCh {
			buf := &bytes.Buffer{}
			assert.Equal(t, 0, buf.Len())
			assert.Equal(t, 0, l.reader.Buffered())
			assert.NotNil(t, raw)
		}
	}()

	go func() {
		for err := range errCh {
			t.Log(err)
			t.Fail()
		}
	}()

	// create a pipe simulating an io.ReadCloser
	reader, writer := io.Pipe()

	// create a buffer that will write to the pipe
	bufferedWriter := bufio.NewWriter(writer)

	// wire the buffers to the logstream
	l.stream = io.NopCloser(reader)
	l.reader = bufio.NewReader(l.stream)

	// write to the pipe periodically
	ticker := time.NewTicker(5 * time.Millisecond)
	endTime := time.Now().Add(1* time.Second)
	go func() {
		for {
			select {
			case<-ticker.C:
				// write data into the buffer
				bufferedWriter.Write([]byte("Hello, world!\n"))
				bufferedWriter.Flush()

				if time.Now().After(endTime) {
					ticker.Stop()
					writer.Close()
					break
				}
			}
		}
	}()

	l.Read()

	close(rawLogCh)
	close(errCh)
}

