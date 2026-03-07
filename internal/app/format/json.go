package format

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/aartinian/nettrace/internal/app"
	"github.com/aartinian/nettrace/internal/trace"
	"github.com/aartinian/nettrace/internal/util"
)

type timingJSON struct {
	DNSMS      int64 `json:"dns_ms"`
	TCPMS      int64 `json:"tcp_ms"`
	TLSMS      int64 `json:"tls_ms"`
	TTFBMS     int64 `json:"ttfb_ms"`
	DownloadMS int64 `json:"download_ms"`
	TotalMS    int64 `json:"total_ms"`
}

type resultJSON struct {
	URL           string     `json:"url"`
	Status        int        `json:"status"`
	Protocol      string     `json:"protocol"`
	Remote        string     `json:"remote,omitempty"`
	TLSVersion    string     `json:"tls_version,omitempty"`
	TLSCipher     string     `json:"tls_cipher,omitempty"`
	Timings       timingJSON `json:"timings"`
	BytesReceived int64      `json:"bytes_received"`
	Redirects     int        `json:"redirects"`
}

type repeatJSON struct {
	URL    string       `json:"url"`
	Repeat int          `json:"repeat"`
	Runs   []resultJSON `json:"runs"`
	Stats  *statsJSON   `json:"stats,omitempty"`
}

type statsJSON struct {
	MinMS int64 `json:"min_ms"`
	MaxMS int64 `json:"max_ms"`
	AvgMS int64 `json:"avg_ms"`
	P95MS int64 `json:"p95_ms"`
}

// RenderJSON writes a JSON summary payload to w.
func RenderJSON(w io.Writer, summary app.Summary) error {
	if len(summary.Runs) == 0 {
		return fmt.Errorf("no trace results to format")
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")

	if len(summary.Runs) == 1 {
		return encoder.Encode(convertResult(summary.Runs[0]))
	}

	runs := make([]resultJSON, 0, len(summary.Runs))
	for _, run := range summary.Runs {
		runs = append(runs, convertResult(run))
	}

	payload := repeatJSON{
		URL:    summary.URL,
		Repeat: len(summary.Runs),
		Runs:   runs,
	}
	if summary.Stats != nil {
		payload.Stats = &statsJSON{
			MinMS: util.DurationMilliseconds(summary.Stats.Min),
			MaxMS: util.DurationMilliseconds(summary.Stats.Max),
			AvgMS: util.DurationMilliseconds(summary.Stats.Avg),
			P95MS: util.DurationMilliseconds(summary.Stats.P95),
		}
	}

	return encoder.Encode(payload)
}

func convertResult(result trace.Result) resultJSON {
	return resultJSON{
		URL:           result.URL,
		Status:        result.StatusCode,
		Protocol:      result.Protocol,
		Remote:        result.RemoteAddr,
		TLSVersion:    result.TLSVersion,
		TLSCipher:     result.TLSCipher,
		Timings:       convertTimings(result.Timings),
		BytesReceived: result.BytesReceived,
		Redirects:     result.Redirects,
	}
}

func convertTimings(timings trace.Timings) timingJSON {
	return timingJSON{
		DNSMS:      util.DurationMilliseconds(timings.DNS),
		TCPMS:      util.DurationMilliseconds(timings.TCPConnect),
		TLSMS:      util.DurationMilliseconds(timings.TLSHandshake),
		TTFBMS:     util.DurationMilliseconds(timings.TTFB),
		DownloadMS: util.DurationMilliseconds(timings.Download),
		TotalMS:    util.DurationMilliseconds(timings.Total),
	}
}
