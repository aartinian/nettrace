package trace

import "time"

func between(start, end time.Time) time.Duration {
	if start.IsZero() || end.IsZero() || end.Before(start) {
		return 0
	}
	return end.Sub(start)
}

func dnsDuration(attempt traceAttempt) time.Duration {
	return between(attempt.dnsStart, attempt.dnsDone)
}
