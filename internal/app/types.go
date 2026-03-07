// Package app orchestrates configuration validation, request execution, and
// result aggregation for the nettrace CLI.
package app

import (
	"net/http"
	"sort"
	"time"

	"github.com/aartinian/nettrace/internal/trace"
)

// Config contains user-facing CLI options translated into execution settings.
type Config struct {
	URL            string
	Method         string
	Headers        http.Header
	Timeout        time.Duration
	ConnectTimeout time.Duration
	Redirects      int
	Repeat         int
	JSON           bool
	Insecure       bool
	NoKeepAlive    bool
}

// Summary is the final result produced for one CLI invocation.
type Summary struct {
	URL   string
	Runs  []trace.Result
	Stats *LatencyStats
}

// LatencyStats stores aggregate values for total request latency.
type LatencyStats struct {
	Min time.Duration
	Max time.Duration
	Avg time.Duration
	P95 time.Duration
}

func buildLatencyStats(results []trace.Result) *LatencyStats {
	if len(results) < 2 {
		return nil
	}

	totals := make([]time.Duration, 0, len(results))
	var sum time.Duration
	for _, result := range results {
		totals = append(totals, result.Timings.Total)
		sum += result.Timings.Total
	}
	sort.Slice(totals, func(i, j int) bool {
		return totals[i] < totals[j]
	})

	idx := (95*len(totals) + 99) / 100
	if idx < 1 {
		idx = 1
	}
	if idx > len(totals) {
		idx = len(totals)
	}

	return &LatencyStats{
		Min: totals[0],
		Max: totals[len(totals)-1],
		Avg: sum / time.Duration(len(totals)),
		P95: totals[idx-1],
	}
}
