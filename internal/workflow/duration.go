package workflow

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

var unitSizes = map[string]time.Duration{
	"s": time.Second,
	"m": time.Minute,
	"h": time.Hour,
	"d": 24 * time.Hour,
	"w": 7 * 24 * time.Hour,
}

func ParseDuration(text string) (time.Duration, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, fmt.Errorf("empty duration")
	}
	var total time.Duration
	rest := text
	for rest != "" {
		i := 0
		for i < len(rest) && rest[i] >= '0' && rest[i] <= '9' {
			i++
		}
		if i == 0 || i == len(rest) {
			return 0, fmt.Errorf("invalid duration %q", text)
		}
		n, err := strconv.Atoi(rest[:i])
		if err != nil {
			return 0, fmt.Errorf("invalid duration %q", text)
		}
		unit, ok := unitSizes[rest[i:i+1]]
		if !ok {
			return 0, fmt.Errorf("invalid duration %q: unknown unit %q", text, rest[i:i+1])
		}
		total += time.Duration(n) * unit
		rest = rest[i+1:]
	}
	if total <= 0 {
		return 0, fmt.Errorf("duration %q must be positive", text)
	}
	return total, nil
}

func FormatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	days := int(d / (24 * time.Hour))
	d -= time.Duration(days) * 24 * time.Hour
	hours := int(d / time.Hour)
	d -= time.Duration(hours) * time.Hour
	minutes := int(d / time.Minute)
	d -= time.Duration(minutes) * time.Minute
	seconds := int(d / time.Second)
	switch {
	case days > 0:
		return fmt.Sprintf("%dd %dh", days, hours)
	case hours > 0:
		return fmt.Sprintf("%dh %dm", hours, minutes)
	case minutes > 0:
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	default:
		return fmt.Sprintf("%ds", seconds)
	}
}

func CompactDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	var b strings.Builder
	for _, u := range []struct {
		suffix string
		size   time.Duration
	}{{"d", 24 * time.Hour}, {"h", time.Hour}, {"m", time.Minute}, {"s", time.Second}} {
		if n := int(d / u.size); n > 0 {
			fmt.Fprintf(&b, "%d%s", n, u.suffix)
			d -= time.Duration(n) * u.size
		}
	}
	if b.Len() == 0 {
		return "1s"
	}
	return b.String()
}
