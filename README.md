# nettrace

`nettrace` is a CLI that breaks down HTTP request latency so you can see where time is spent: DNS, TCP connect, TLS handshake, server time (TTFB), and response download.

## Why nettrace exists

APIs can be slow for very different reasons. `nettrace` helps answer:

- Is DNS slow?
- Is connection setup slow?
- Is TLS negotiation expensive?
- Is the server slow to respond?
- Is payload download the bottleneck?

`nettrace` uses Go's `httptrace` instrumentation and reports measured timings.

## Requirements

- Go 1.24+

## Installation

### From source

```bash
go install github.com/aartinian/nettrace/cmd/nettrace@latest
```

### Local build

```bash
go build ./cmd/nettrace
./nettrace https://example.com
```

### Install script

```bash
./scripts/install.sh
```

## Quick Start

```bash
go build ./cmd/nettrace
./nettrace https://example.com
```

## Usage

```bash
nettrace [flags] <url>
```

Flags:

- `--method string` HTTP method (default `GET`)
- `--header "K: V"` request header (repeatable)
- `--timeout duration` total timeout (default `10s`)
- `--connect-timeout duration` TCP connect timeout (default `3s`)
- `--redirects int` max redirects (default `3`)
- `--repeat int` repeat request N times
- `--json` output JSON
- `--insecure` skip TLS verification
- `--no-keepalive` disable connection reuse
- `--version` print version

## Architecture

`nettrace` keeps responsibilities small and isolated:

- `cmd/nettrace`: CLI entrypoint and flag parsing.
- `internal/app`: orchestration and config validation.
- `internal/trace`: HTTP client setup and `httptrace` instrumentation.
- `internal/app/format`: human and JSON rendering.
- `internal/util`: shared formatting and error helpers.

This structure keeps measurement logic separate from presentation and CLI wiring.

## Examples

```bash
nettrace https://api.example.com/health
```

```bash
nettrace https://api.example.com --repeat 5
```

```bash
nettrace https://api.example.com --json
```

Example output:

```text
HTTP     200 (HTTP/2.0)
Remote   104.18.21.4:443
TLS      TLS1.3 AES_128_GCM_SHA256

DNS               12 ms
TCP connect       18 ms
TLS handshake     24 ms
TTFB             210 ms
Download           3 ms
Total            269 ms

Bytes received 1.2 KB
Redirects      0
```

## Output explanation

- `DNS`: name resolution latency.
- `TCP connect`: socket connect latency.
- `TLS handshake`: TLS negotiation latency (HTTPS only).
- `TTFB`: time from request write to first response byte.
- `Download`: time from first byte to fully reading the response body.
- `Total`: full request lifecycle including redirects.

Additional metadata includes HTTP status, protocol, remote address, TLS details, bytes received, and redirect count.

When using `--repeat N`, nettrace also reports total-latency summary statistics: min, max, avg, and p95.


## Development

```bash
go mod tidy
go vet ./...
go test ./...
go build ./...
```

CI runs vet, tests, linting, and build on each push and pull request.
