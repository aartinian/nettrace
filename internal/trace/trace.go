// Package trace measures HTTP request phase timings using net/http/httptrace.
//
// It captures DNS lookup, TCP connect, TLS handshake, TTFB, download, and
// total time across the full request lifecycle (including redirects).
package trace

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"net/http/httptrace"
	"sync"
	"time"
)

// Timings contains measured phase durations for one request lifecycle.
type Timings struct {
	DNS          time.Duration
	TCPConnect   time.Duration
	TLSHandshake time.Duration
	TTFB         time.Duration
	Download     time.Duration
	Total        time.Duration
}

// Result is the full trace payload for one executed request.
type Result struct {
	URL           string
	StatusCode    int
	Protocol      string
	RemoteAddr    string
	TLSVersion    string
	TLSCipher     string
	BytesReceived int64
	Redirects     int
	Timings       Timings
}

// Tracer executes HTTP requests and records phase timing data.
type Tracer struct {
	transport *http.Transport
	client    *http.Client
}

type collectorKey struct{}

// NewTracer builds a Tracer with a configured HTTP client and transport.
func NewTracer(cfg ClientConfig) *Tracer {
	transport := newHTTPTransport(cfg)
	tracingRT := &tracingRoundTripper{base: transport}

	return &Tracer{
		transport: transport,
		client:    newHTTPClient(cfg, tracingRT),
	}
}

// Close releases idle resources held by the underlying transport.
func (t *Tracer) Close() {
	if t == nil || t.transport == nil {
		return
	}
	t.transport.CloseIdleConnections()
}

// Trace executes the request and returns measured timing and metadata fields.
func (t *Tracer) Trace(request *http.Request) (Result, error) {
	collector := newTraceCollector()

	ctx := context.WithValue(request.Context(), collectorKey{}, collector)
	request = request.WithContext(ctx)

	response, err := t.client.Do(request)
	if response != nil && response.Body != nil {
		_, copyErr := io.Copy(io.Discard, response.Body)
		closeErr := response.Body.Close()
		if copyErr != nil {
			return Result{}, copyErr
		}
		if closeErr != nil {
			return Result{}, closeErr
		}
	}
	if err != nil {
		return Result{}, err
	}

	snapshot := collector.snapshot()
	result := Result{
		URL:           response.Request.URL.String(),
		StatusCode:    response.StatusCode,
		Protocol:      response.Proto,
		RemoteAddr:    snapshot.remoteAddr,
		BytesReceived: snapshot.bytesReceived,
		Redirects:     snapshot.redirects,
		Timings:       snapshot.timings,
	}
	if response.TLS != nil {
		result.TLSVersion = TLSVersionName(response.TLS.Version)
		result.TLSCipher = TLSCipherName(response.TLS.CipherSuite)
	}

	return result, nil
}

type traceAttempt struct {
	requestStart time.Time

	dnsStart time.Time
	dnsDone  time.Time

	connectStart time.Time
	connectDone  time.Time

	tlsStart time.Time
	tlsDone  time.Time

	wroteRequest time.Time
	firstByte    time.Time
	bodyDone     time.Time

	remoteAddr string
	bytesRead  int64
}

type traceSnapshot struct {
	timings       Timings
	remoteAddr    string
	bytesReceived int64
	redirects     int
}

type traceCollector struct {
	mu       sync.Mutex
	attempts []*traceAttempt
}

func newTraceCollector() *traceCollector {
	return &traceCollector{}
}

func (c *traceCollector) beginAttempt() *traceAttempt {
	c.mu.Lock()
	defer c.mu.Unlock()

	attempt := &traceAttempt{requestStart: time.Now()}
	c.attempts = append(c.attempts, attempt)
	return attempt
}

func (c *traceCollector) roundTrip(base http.RoundTripper, request *http.Request) (*http.Response, error) {
	attempt := c.beginAttempt()

	trace := &httptrace.ClientTrace{
		DNSStart: func(httptrace.DNSStartInfo) {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.dnsStart = time.Now()
			})
		},
		DNSDone: func(httptrace.DNSDoneInfo) {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.dnsDone = time.Now()
			})
		},
		ConnectStart: func(_, _ string) {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.connectStart = time.Now()
			})
		},
		ConnectDone: func(_, _ string, _ error) {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.connectDone = time.Now()
			})
		},
		TLSHandshakeStart: func() {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.tlsStart = time.Now()
			})
		},
		TLSHandshakeDone: func(tls.ConnectionState, error) {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.tlsDone = time.Now()
			})
		},
		WroteRequest: func(httptrace.WroteRequestInfo) {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.wroteRequest = time.Now()
			})
		},
		GotConn: func(info httptrace.GotConnInfo) {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.remoteAddr = remoteAddrFromConn(info.Conn)
			})
		},
		GotFirstResponseByte: func() {
			c.updateAttempt(attempt, func(a *traceAttempt) {
				a.firstByte = time.Now()
			})
		},
	}

	request = request.WithContext(httptrace.WithClientTrace(request.Context(), trace))
	response, err := base.RoundTrip(request)
	if err != nil {
		c.completeAttempt(attempt, 0)
		return response, err
	}

	if response == nil || response.Body == nil {
		c.completeAttempt(attempt, 0)
		return response, nil
	}

	response.Body = newTimingBody(response.Body, func(bytesRead int64) {
		c.completeAttempt(attempt, bytesRead)
	})

	return response, nil
}

func (c *traceCollector) updateAttempt(attempt *traceAttempt, fn func(*traceAttempt)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	fn(attempt)
}

func (c *traceCollector) completeAttempt(attempt *traceAttempt, bytesRead int64) {
	c.updateAttempt(attempt, func(a *traceAttempt) {
		a.bytesRead = bytesRead
		a.bodyDone = time.Now()
	})
}

func (c *traceCollector) snapshot() traceSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()

	snapshot := traceSnapshot{}
	if len(c.attempts) == 0 {
		return snapshot
	}

	firstStart := c.attempts[0].requestStart
	lastEnd := c.attempts[0].bodyDone

	for _, attempt := range c.attempts {
		snapshot.timings.DNS += dnsDuration(*attempt)
		snapshot.timings.TCPConnect += connectDuration(*attempt)
		snapshot.timings.TLSHandshake += tlsDuration(*attempt)
		snapshot.timings.TTFB += between(attempt.wroteRequest, attempt.firstByte)
		snapshot.timings.Download += between(attempt.firstByte, attempt.bodyDone)
		snapshot.bytesReceived += attempt.bytesRead

		if attempt.remoteAddr != "" {
			snapshot.remoteAddr = attempt.remoteAddr
		}
		if !attempt.requestStart.IsZero() && attempt.requestStart.Before(firstStart) {
			firstStart = attempt.requestStart
		}
		if !attempt.bodyDone.IsZero() && attempt.bodyDone.After(lastEnd) {
			lastEnd = attempt.bodyDone
		}
	}

	snapshot.timings.Total = between(firstStart, lastEnd)
	if len(c.attempts) > 1 {
		snapshot.redirects = len(c.attempts) - 1
	}

	return snapshot
}

type tracingRoundTripper struct {
	base http.RoundTripper
}

func (t *tracingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	collector, _ := request.Context().Value(collectorKey{}).(*traceCollector)
	if collector == nil {
		return t.base.RoundTrip(request)
	}

	return collector.roundTrip(t.base, request)
}

type timingBody struct {
	body   io.ReadCloser
	onDone func(bytesRead int64)

	mu        sync.Mutex
	bytesRead int64
	completed bool
}

func newTimingBody(body io.ReadCloser, onDone func(int64)) *timingBody {
	return &timingBody{
		body:   body,
		onDone: onDone,
	}
}

func (b *timingBody) Read(p []byte) (int, error) {
	n, err := b.body.Read(p)

	b.mu.Lock()
	b.bytesRead += int64(n)
	b.mu.Unlock()

	if err == io.EOF {
		b.finish()
	}

	return n, err
}

func (b *timingBody) Close() error {
	err := b.body.Close()
	b.finish()
	return err
}

func (b *timingBody) finish() {
	b.mu.Lock()
	if b.completed {
		b.mu.Unlock()
		return
	}
	b.completed = true
	bytesRead := b.bytesRead
	b.mu.Unlock()

	if b.onDone != nil {
		b.onDone(bytesRead)
	}
}
