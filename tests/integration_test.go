package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/aartinian/nettrace/internal/app"
)

func TestHTTPRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(server.Close)

	summary, err := app.Execute(context.Background(), baseConfig(server.URL))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	if len(summary.Runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(summary.Runs))
	}

	result := summary.Runs[0]
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want %d", result.StatusCode, http.StatusOK)
	}
	if !strings.HasPrefix(result.Protocol, "HTTP/") {
		t.Fatalf("unexpected protocol: %s", result.Protocol)
	}
	if result.Timings.Total <= 0 {
		t.Fatalf("expected total timing > 0, got %s", result.Timings.Total)
	}
}

func TestHTTPSRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secure"))
	}))
	t.Cleanup(server.Close)

	cfg := baseConfig(server.URL)
	cfg.Insecure = true

	summary, err := app.Execute(context.Background(), cfg)
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := summary.Runs[0]
	if result.TLSVersion == "" {
		t.Fatalf("expected TLS version for HTTPS request")
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want %d", result.StatusCode, http.StatusOK)
	}
}

func TestRedirects(t *testing.T) {
	t.Parallel()

	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("destination"))
	}))
	t.Cleanup(target.Close)

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(redirector.Close)

	summary, err := app.Execute(context.Background(), baseConfig(redirector.URL))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := summary.Runs[0]
	if result.Redirects != 1 {
		t.Fatalf("redirects=%d, want 1", result.Redirects)
	}
	if result.StatusCode != http.StatusOK {
		t.Fatalf("status=%d, want %d", result.StatusCode, http.StatusOK)
	}
}

func TestSlowResponseTiming(t *testing.T) {
	t.Parallel()

	const delay = 80 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		_, _ = w.Write([]byte("slow"))
	}))
	t.Cleanup(server.Close)

	summary, err := app.Execute(context.Background(), baseConfig(server.URL))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := summary.Runs[0]
	if result.Timings.TTFB < 60*time.Millisecond {
		t.Fatalf("ttfb=%s, expected at least 60ms", result.Timings.TTFB)
	}
	if result.Timings.Total < result.Timings.TTFB {
		t.Fatalf("total=%s should be >= ttfb=%s", result.Timings.Total, result.Timings.TTFB)
	}
}

func TestStreamedResponseBodyTiming(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "flusher unavailable", http.StatusInternalServerError)
			return
		}

		for i := 0; i < 3; i++ {
			_, _ = fmt.Fprintf(w, "chunk-%d\n", i)
			flusher.Flush()
			time.Sleep(20 * time.Millisecond)
		}
	}))
	t.Cleanup(server.Close)

	summary, err := app.Execute(context.Background(), baseConfig(server.URL))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}

	result := summary.Runs[0]
	if result.BytesReceived == 0 {
		t.Fatalf("expected non-zero bytes received")
	}
	if result.Timings.Download <= 0 {
		t.Fatalf("expected download timing > 0, got %s", result.Timings.Download)
	}
}

func baseConfig(url string) app.Config {
	return app.Config{
		URL:            url,
		Method:         http.MethodGet,
		Headers:        make(http.Header),
		Timeout:        5 * time.Second,
		ConnectTimeout: 2 * time.Second,
		Redirects:      3,
		Repeat:         1,
	}
}
