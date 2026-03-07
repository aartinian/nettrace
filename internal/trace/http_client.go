package trace

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"
)

// ClientConfig controls how requests are executed.
type ClientConfig struct {
	Timeout        time.Duration
	ConnectTimeout time.Duration
	MaxRedirects   int
	Insecure       bool
	NoKeepAlive    bool
}

// newHTTPTransport creates the base transport used by the tracer client.
func newHTTPTransport(cfg ClientConfig) *http.Transport {
	dialer := &net.Dialer{
		Timeout:   cfg.ConnectTimeout,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		DisableKeepAlives:     cfg.NoKeepAlive,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   cfg.ConnectTimeout,
		ExpectContinueTimeout: time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: cfg.Insecure,
		},
	}
}

// newHTTPClient creates the HTTP client with timeout and redirect behavior.
func newHTTPClient(cfg ClientConfig, transport http.RoundTripper) *http.Client {
	client := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}

	client.CheckRedirect = func(_ *http.Request, via []*http.Request) error {
		if len(via) > cfg.MaxRedirects {
			return fmt.Errorf("stopped after %d redirects", cfg.MaxRedirects)
		}
		return nil
	}

	return client
}
