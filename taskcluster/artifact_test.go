package taskcluster

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetHttpResponseCappedReturnsFullBodyUnderCap(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("hello world"))
	}))
	defer server.Close()

	content, contentType, truncated, err := getHttpResponseCapped(server.URL, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "hello world" {
		t.Fatalf("unexpected content: %q", content)
	}
	if contentType != "text/plain" {
		t.Fatalf("unexpected content type: %q", contentType)
	}
	if truncated {
		t.Fatalf("expected truncated=false for a body under the cap")
	}
}

func TestGetHttpResponseCappedTruncatesOversizedBody(t *testing.T) {
	body := strings.Repeat("x", 1000)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	defer server.Close()

	content, _, truncated, err := getHttpResponseCapped(server.URL, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) != 100 {
		t.Fatalf("expected content capped at 100 bytes, got %d", len(content))
	}
	if !truncated {
		t.Fatalf("expected truncated=true for a body over the cap")
	}
}

func TestGetHttpResponseCappedFollowsRedirect(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("redirected content"))
	}))
	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusSeeOther)
	}))
	defer redirector.Close()

	content, _, truncated, err := getHttpResponseCapped(redirector.URL, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(content) != "redirected content" {
		t.Fatalf("unexpected content: %q", content)
	}
	if truncated {
		t.Fatalf("expected truncated=false")
	}
}
