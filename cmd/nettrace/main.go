// Command nettrace measures where HTTP request latency is spent.
//
// It performs a real request and reports phase timings (DNS, TCP connect,
// TLS handshake, TTFB, download, total) using Go's httptrace instrumentation.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aartinian/nettrace/internal/app"
	appformat "github.com/aartinian/nettrace/internal/app/format"
	"github.com/aartinian/nettrace/internal/util"
)

var version = "dev"

const (
	exitCodeOK    = 0
	exitCodeError = 1
	exitCodeUsage = 2

	defaultTimeout        = 10 * time.Second
	defaultConnectTimeout = 3 * time.Second
	defaultRedirects      = 3
	defaultRepeat         = 1
)

// headerValues implements flag.Value for repeatable --header flags.
type headerValues struct {
	values http.Header
}

func (h *headerValues) String() string {
	if h == nil || len(h.values) == 0 {
		return ""
	}

	parts := make([]string, 0, len(h.values))
	for key, vals := range h.values {
		for _, value := range vals {
			parts = append(parts, fmt.Sprintf("%s: %s", key, value))
		}
	}

	return strings.Join(parts, ", ")
}

func (h *headerValues) Set(raw string) error {
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid header %q, expected format 'K: V'", raw)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])
	if key == "" {
		return fmt.Errorf("invalid header %q, header name cannot be empty", raw)
	}

	if h.values == nil {
		h.values = make(http.Header)
	}
	h.values.Add(key, value)

	return nil
}

func (h *headerValues) Header() http.Header {
	if h.values == nil {
		return make(http.Header)
	}

	clone := make(http.Header, len(h.values))
	for key, vals := range h.values {
		copied := make([]string, len(vals))
		copy(copied, vals)
		clone[key] = copied
	}

	return clone
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run executes one CLI invocation and returns a process exit code.
func run(args []string, stdout io.Writer, stderr io.Writer) int {
	cfg, showVersion, showHelp, err := parseConfig(args)
	if err != nil {
		var usageErr *util.UsageError
		if errors.As(err, &usageErr) {
			if _, writeErr := fmt.Fprintf(stderr, "Error: %s\n\n", err); writeErr != nil {
				return exitCodeError
			}
			if writeErr := printUsage(stderr); writeErr != nil {
				return exitCodeError
			}
			return exitCodeUsage
		}

		if _, writeErr := fmt.Fprintf(stderr, "Error: %v\n", err); writeErr != nil {
			return exitCodeError
		}
		return exitCodeError
	}
	if showHelp {
		if err := printUsage(stdout); err != nil {
			return exitCodeError
		}
		return exitCodeOK
	}

	if showVersion {
		if _, err := fmt.Fprintln(stdout, version); err != nil {
			return exitCodeError
		}
		return exitCodeOK
	}

	summary, err := app.Execute(context.Background(), cfg)
	if err != nil {
		var usageErr *util.UsageError
		if errors.As(err, &usageErr) {
			if _, writeErr := fmt.Fprintf(stderr, "Error: %s\n\n", err); writeErr != nil {
				return exitCodeError
			}
			if writeErr := printUsage(stderr); writeErr != nil {
				return exitCodeError
			}
			return exitCodeUsage
		}

		if _, writeErr := fmt.Fprintf(stderr, "Error: %v\n", err); writeErr != nil {
			return exitCodeError
		}
		return exitCodeError
	}

	if cfg.JSON {
		err = appformat.RenderJSON(stdout, summary)
	} else {
		err = appformat.RenderTable(stdout, summary)
	}
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "Error: %v\n", err); writeErr != nil {
			return exitCodeError
		}
		return exitCodeError
	}

	return exitCodeOK
}

// parseConfig parses CLI flags and positional arguments into app.Config.
func parseConfig(args []string) (app.Config, bool, bool, error) {
	fs := flag.NewFlagSet("nettrace", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}

	method := fs.String("method", http.MethodGet, "HTTP method")
	timeout := fs.Duration("timeout", defaultTimeout, "Total timeout")
	connectTimeout := fs.Duration("connect-timeout", defaultConnectTimeout, "TCP connect timeout")
	redirects := fs.Int("redirects", defaultRedirects, "Max redirects")
	repeat := fs.Int("repeat", defaultRepeat, "Repeat request N times")
	jsonOutput := fs.Bool("json", false, "Output JSON")
	insecure := fs.Bool("insecure", false, "Skip TLS verification")
	noKeepalive := fs.Bool("no-keepalive", false, "Disable connection reuse")
	showVersion := fs.Bool("version", false, "Print version")

	var headers headerValues
	fs.Var(&headers, "header", "Request header (repeatable)")

	var targetURL string
	parseArgs := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		targetURL = args[0]
		parseArgs = args[1:]
	}

	if err := fs.Parse(parseArgs); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return app.Config{}, false, true, nil
		}
		return app.Config{}, false, false, util.NewUsageError("%s", err)
	}

	if targetURL == "" {
		extra := fs.Args()
		switch len(extra) {
		case 0:
			if !*showVersion {
				return app.Config{}, false, false, util.NewUsageError("URL is required")
			}
		case 1:
			targetURL = extra[0]
		default:
			return app.Config{}, false, false, util.NewUsageError("too many positional arguments")
		}
	} else if len(fs.Args()) > 0 {
		return app.Config{}, false, false, util.NewUsageError("unexpected positional arguments: %s", strings.Join(fs.Args(), " "))
	}

	cfg := app.Config{
		URL:            targetURL,
		Method:         *method,
		Headers:        headers.Header(),
		Timeout:        *timeout,
		ConnectTimeout: *connectTimeout,
		Redirects:      *redirects,
		Repeat:         *repeat,
		JSON:           *jsonOutput,
		Insecure:       *insecure,
		NoKeepAlive:    *noKeepalive,
	}

	return cfg, *showVersion, false, nil
}

// printUsage prints CLI help text.
func printUsage(w io.Writer) error {
	_, err := io.WriteString(w, `Usage:
  nettrace [flags] <url>

Examples:
  nettrace https://example.com
  nettrace --repeat 5 --json https://api.example.com/health

Flags:
  --method string         HTTP method (default GET)
  --header "K: V"        Request header (repeatable)
  --timeout duration      Total timeout (default 10s)
  --connect-timeout duration
                          TCP connect timeout (default 3s)
  --redirects int         Max redirects (default 3)
  --repeat int            Repeat request N times
  --json                  Output JSON
  --insecure              Skip TLS verification
  --no-keepalive          Disable connection reuse
  --version               Print version
`)
	return err
}
