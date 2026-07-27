package observability

import (
	"bufio"
	"net"
	"net/http"
	"testing"
)

type metricsHijackWriter struct {
	header   http.Header
	hijacked bool
}

func (w *metricsHijackWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (*metricsHijackWriter) Write(payload []byte) (int, error) { return len(payload), nil }
func (*metricsHijackWriter) WriteHeader(int)                   {}

func (w *metricsHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	server, client := net.Pipe()
	_ = client.Close()
	return server, bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)), nil
}

func TestStatusRecorderPreservesHijacker(t *testing.T) {
	underlying := &metricsHijackWriter{}
	recorder := &statusRecorder{ResponseWriter: underlying, status: http.StatusOK}

	connection, _, err := recorder.Hijack()
	if err != nil {
		t.Fatalf("Hijack() error = %v", err)
	}
	defer connection.Close()
	if !underlying.hijacked {
		t.Fatal("underlying Hijack was not called")
	}
	if recorder.status != http.StatusSwitchingProtocols {
		t.Fatalf("status = %d, want %d", recorder.status, http.StatusSwitchingProtocols)
	}
	if recorder.Unwrap() != underlying {
		t.Fatal("Unwrap() did not return the underlying writer")
	}
}
