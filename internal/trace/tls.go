package trace

import (
	"crypto/tls"
	"strings"
	"time"
)

func tlsDuration(attempt traceAttempt) time.Duration {
	return between(attempt.tlsStart, attempt.tlsDone)
}

// TLSVersionName returns a compact user-facing TLS version string.
func TLSVersionName(version uint16) string {
	switch version {
	case tls.VersionTLS10:
		return "TLS1.0"
	case tls.VersionTLS11:
		return "TLS1.1"
	case tls.VersionTLS12:
		return "TLS1.2"
	case tls.VersionTLS13:
		return "TLS1.3"
	default:
		return ""
	}
}

// TLSCipherName returns a user-facing cipher suite name without TLS_ prefixes.
func TLSCipherName(cipherSuite uint16) string {
	name := tls.CipherSuiteName(cipherSuite)
	name = strings.TrimPrefix(name, "TLS_")
	name = strings.TrimPrefix(name, "TLS13_")
	return name
}
