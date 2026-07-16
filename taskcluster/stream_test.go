package taskcluster

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// flushWrite writes s and flushes it out as its own chunk.
func flushWrite(w http.ResponseWriter, s string) {
	w.Write([]byte(s))
	w.(http.Flusher).Flush()
}

func TestStreamHttpResponseDeliversChunksProgressively(t *testing.T) {
	firstChunkSeen := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		flushWrite(w, "one\n")
		// Hold the stream open until the client has demonstrably received
		// the first chunk while the response is still in flight — the
		// whole point of streaming over a ReadAll.
		select {
		case <-firstChunkSeen:
		case <-time.After(5 * time.Second):
			t.Error("client never saw the first chunk while the stream was open")
		}
		flushWrite(w, "two\n")
	}))
	defer srv.Close()

	var got strings.Builder
	first := true
	contentType, truncated, err := streamHttpResponse(srv.URL, 1024, nil, func(chunk []byte) {
		got.Write(chunk)
		if first {
			first = false
			close(firstChunkSeen)
		}
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if truncated {
		t.Fatal("unexpected truncation")
	}
	if contentType != "text/plain" {
		t.Fatalf("unexpected content type: %q", contentType)
	}
	if got.String() != "one\ntwo\n" {
		t.Fatalf("unexpected content: %q", got.String())
	}
}

func TestStreamHttpResponseStop(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flushWrite(w, "start\n")
		// Never write again; hold the stream open until the client hangs up
		// (as a live log does while the task keeps running).
		<-r.Context().Done()
	}))
	defer srv.Close()

	stop := make(chan struct{})
	var got strings.Builder
	done := make(chan error, 1)
	go func() {
		_, _, err := streamHttpResponse(srv.URL, 1024, stop, func(chunk []byte) {
			got.Write(chunk)
			close(stop) // stop as soon as the first chunk lands
		})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected a clean return after stop, got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("streamHttpResponse did not return after stop was closed")
	}
	if got.String() != "start\n" {
		t.Fatalf("unexpected content: %q", got.String())
	}
}

func TestStreamHttpResponseCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flushWrite(w, strings.Repeat("x", 100))
	}))
	defer srv.Close()

	var got strings.Builder
	_, truncated, err := streamHttpResponse(srv.URL, 40, nil, func(chunk []byte) {
		got.Write(chunk)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !truncated {
		t.Fatal("expected truncation at the cap")
	}
	if got.Len() != 40 {
		t.Fatalf("expected exactly 40 bytes, got %d", got.Len())
	}
}

func TestStreamHttpResponseConnectError(t *testing.T) {
	// A port nothing listens on — the initial GET itself must fail.
	if _, _, err := streamHttpResponse("http://127.0.0.1:1/nope", 1024, nil, func([]byte) {}); err == nil {
		t.Fatal("expected a connection error")
	}
}
