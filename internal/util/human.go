package util

import (
	"fmt"
	"time"
)

// DurationMilliseconds returns non-negative milliseconds for display/JSON use.
func DurationMilliseconds(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	return d.Milliseconds()
}

// FormatDuration renders a duration as "<ms> ms".
func FormatDuration(d time.Duration) string {
	return fmt.Sprintf("%d ms", DurationMilliseconds(d))
}

// FormatBytes renders bytes using binary units (KB, MB, ...).
func FormatBytes(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(bytes)
	for _, unit := range units {
		value /= 1024
		if value < 1024 || unit == units[len(units)-1] {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%d B", bytes)
}
