package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/aartinian/nettrace/internal/trace"
)

// Execute validates config, runs one or more traces, and returns a summary.
func Execute(ctx context.Context, cfg Config) (Summary, error) {
	if err := ValidateConfig(cfg); err != nil {
		return Summary{}, err
	}

	tracer := trace.NewTracer(trace.ClientConfig{
		Timeout:        cfg.Timeout,
		ConnectTimeout: cfg.ConnectTimeout,
		MaxRedirects:   cfg.Redirects,
		Insecure:       cfg.Insecure,
		NoKeepAlive:    cfg.NoKeepAlive,
	})
	defer tracer.Close()

	runs := make([]trace.Result, 0, cfg.Repeat)
	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	for i := 0; i < cfg.Repeat; i++ {
		request, err := http.NewRequestWithContext(ctx, method, cfg.URL, nil)
		if err != nil {
			return Summary{}, fmt.Errorf("build request: %w", err)
		}
		request.Header = cloneHeaders(cfg.Headers)

		result, err := tracer.Trace(request)
		if err != nil {
			return Summary{}, err
		}

		runs = append(runs, result)
	}

	summary := Summary{
		URL:  cfg.URL,
		Runs: runs,
	}
	if cfg.Repeat > 1 {
		summary.Stats = buildLatencyStats(runs)
	}

	return summary, nil
}

// cloneHeaders creates a deep copy to avoid cross-run header mutation.
func cloneHeaders(headers http.Header) http.Header {
	if len(headers) == 0 {
		return make(http.Header)
	}

	clone := make(http.Header, len(headers))
	for key, values := range headers {
		copied := make([]string, len(values))
		copy(copied, values)
		clone[key] = copied
	}

	return clone
}
