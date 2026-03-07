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

	if _, err := fmt.Fprintf(w, "HTTP     %d (%s)\n", result.StatusCode, result.Protocol); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Remote   %s\n", valueOrDash(result.RemoteAddr)); err != nil {
		return err
	}
	if result.TLSVersion != "" {
		if result.TLSCipher != "" {
			if _, err := fmt.Fprintf(w, "TLS      %s %s\n", result.TLSVersion, result.TLSCipher); err != nil {
				return err
			}
		} else {
			if _, err := fmt.Fprintf(w, "TLS      %s\n", result.TLSVersion); err != nil {
				return err
			}
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if err := writeTiming(w, "DNS", result.Timings.DNS); err != nil {
		return err
	}
	if err := writeTiming(w, "TCP connect", result.Timings.TCPConnect); err != nil {
		return err
	}
	if err := writeTiming(w, "TLS handshake", result.Timings.TLSHandshake); err != nil {
		return err
	}
	if err := writeTiming(w, "TTFB", result.Timings.TTFB); err != nil {
		return err
	}
	if err := writeTiming(w, "Download", result.Timings.Download); err != nil {
		return err
	}
	if err := writeTiming(w, "Total", result.Timings.Total); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, "Bytes received %s\n", util.FormatBytes(result.BytesReceived)); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "Redirects      %d\n", result.Redirects); err != nil {
		return err
	}

	if summary.Stats != nil {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "Total latency stats (%d runs)\n", len(summary.Runs)); err != nil {
			return err
		}
		if err := writeTiming(w, "min", summary.Stats.Min); err != nil {
			return err
		}
		if err := writeTiming(w, "max", summary.Stats.Max); err != nil {
			return err
		}
		if err := writeTiming(w, "avg", summary.Stats.Avg); err != nil {
			return err
		}
		if err := writeTiming(w, "p95", summary.Stats.P95); err != nil {
			return err
		}
	}

	return nil
}

func writeTiming(w io.Writer, label string, duration time.Duration) error {
	_, err := fmt.Fprintf(w, "%-13s %8s\n", label, util.FormatDuration(duration))
	return err
}

func valueOrDash(value string) string {
	if value == "" {
		return "-"
	}
	return value
}
