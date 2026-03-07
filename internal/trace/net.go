package trace

import (
	"net"
	"time"
)

func connectDuration(attempt traceAttempt) time.Duration {
	return between(attempt.connectStart, attempt.connectDone)
}

func remoteAddrFromConn(conn net.Conn) string {
	if conn == nil || conn.RemoteAddr() == nil {
		return ""
	}

	return conn.RemoteAddr().String()
}
