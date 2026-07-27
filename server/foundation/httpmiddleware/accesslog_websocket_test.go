package httpmiddleware

import (
	"bufio"
	"net"
	"net/http"
	"testing"
)

type accessLogHijackWriter struct {
	header  http.Header
	hijacked bool
}

func (w *accessLogHijackWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (*accessLogHijackWriter) Write(payload []byte) (int, error) { return len(payload), nil }
func (*accessLogHijackWriter) WriteHeader(int)                   {}

func (w *accessLogHijackWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	server, client := net.Pipe()
	_ = client.Close()
	return server, bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)), nil
}

func TestStatusWriterPreservesHijacker(t *testing.T) {
	underlying := &accessLogHijackWriter{}
	writer := &statusWriter{ResponseWriter: underlying}

	connection, _, err := writer.Hijack()
	if err != nil {
		t.Fatalf("Hijack() error = %v", err)
	}
	defer connection.Close()
	if !underlying.hijacked {
		t.Fatal("underlying Hijack was not called")
	}
	if writer.statusCode != http.StatusSwitchingProtocols {
		t.Fatalf("statusCode = %d, want %d", writer.statusCode, http.StatusSwitchingProtocols)
	}
	if writer.Unwrap() != underlying {
		t.Fatal("Unwrap() did not return the underlying writer")
	}
}
