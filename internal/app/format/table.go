// Package format renders trace summaries in human-readable and machine-friendly
// output formats.
package format

import (
	"fmt"
	"io"
	"time"

	"github.com/aartinian/nettrace/internal/app"
	"github.com/aartinian/nettrace/internal/util"
)

// RenderTable writes a human-readable latency report to w.
func RenderTable(w io.Writer, summary app.Summary) error {
	if len(summary.Runs) == 0 {
		return fmt.Errorf("no trace results to format")
	}

	result := summary.Runs[len(summary.Runs)-1]

	fmt.Fprintf(w, "HTTP     %d (%s)\n", result.StatusCode, result.Protocol)
	fmt.Fprintf(w, "Remote   %s\n", valueOrDash(result.RemoteAddr))
	if result.TLSVersion != "" {
		if result.TLSCipher != "" {
			fmt.Fprintf(w, "TLS      %s %s\n", result.TLSVersion, result.TLSCipher)
		} else {
			fmt.Fprintf(w, "TLS      %s\n", result.TLSVersion)
		}
	}
	fmt.Fprintln(w)

	writeTiming(w, "DNS", result.Timings.DNS)
	writeTiming(w, "TCP connect", result.Timings.TCPConnect)
	writeTiming(w, "TLS handshake", result.Timings.TLSHandshake)
	writeTiming(w, "TTFB", result.Timings.TTFB)
	writeTiming(w, "Download", result.Timings.Download)
	writeTiming(w, "Total", result.Timings.Total)
	fmt.Fprintln(w)

	fmt.Fprintf(w, "Bytes received %s\n", util.FormatBytes(result.BytesReceived))
	fmt.Fprintf(w, "Redirects      %d\n", result.Redirects)

	if summary.Stats != nil {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "Total latency stats (%d runs)\n", len(summary.Runs))
		writeTiming(w, "min", summary.Stats.Min)
		writeTiming(w, "max", summary.Stats.Max)
		writeTiming(w, "avg", summary.Stats.Avg)
		writeTiming(w, "p95", summary.Stats.P95)
	}

	return nil
}

func writeTiming(w io.Writer, label string, duration time.Duration) {
	fmt.Fprintf(w, "%-13s %8s\n", label, util.FormatDuration(duration))
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
