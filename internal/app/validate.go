package app

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/aartinian/nettrace/internal/util"
)

// ValidateConfig checks user input before request execution starts.
func ValidateConfig(cfg Config) error {
	if cfg.URL == "" {
		return util.NewUsageError("URL is required")
	}

	parsed, err := url.Parse(cfg.URL)
	if err != nil {
		return util.NewUsageError("invalid URL: %v", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return util.NewUsageError("URL scheme must be http or https")
	}
	if parsed.Host == "" {
		return util.NewUsageError("URL must include a host")
	}

	method := strings.TrimSpace(cfg.Method)
	if method == "" {
		return util.NewUsageError("method cannot be empty")
	}
	if !validHTTPMethod(method) {
		return util.NewUsageError("invalid HTTP method %q", cfg.Method)
	}

	if cfg.Timeout <= 0 {
		return util.NewUsageError("timeout must be greater than zero")
	}
	if cfg.ConnectTimeout <= 0 {
		return util.NewUsageError("connect-timeout must be greater than zero")
	}
	if cfg.Redirects < 0 {
		return util.NewUsageError("redirects cannot be negative")
	}
	if cfg.Repeat < 1 {
		return util.NewUsageError("repeat must be at least 1")
	}

	return nil
}

// validHTTPMethod accepts the standard HTTP methods exposed by net/http.
func validHTTPMethod(method string) bool {
	for _, token := range []string{
		http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodOptions,
		http.MethodTrace,
		http.MethodConnect,
	} {
		if strings.EqualFold(method, token) {
			return true
		}
	}

	return false
}
