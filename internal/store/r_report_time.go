package store

import (
	"fmt"
	"strings"
	"time"
)

// NormalizeReportTimeRange converts UI date / datetime-local values into
// [start, endExclusive) timestamps for report queries.
// Supported inputs: "YYYY-MM-DD", "YYYY-MM-DDTHH:MM", "YYYY-MM-DD HH:MM[:SS]".
func NormalizeReportTimeRange(from, to string) (string, string, error) {
	start, err := parseReportBound(from, false)
	if err != nil {
		return "", "", fmt.Errorf("boshlanish vaqti: %w", err)
	}
	endExclusive, err := parseReportBound(to, true)
	if err != nil {
		return "", "", fmt.Errorf("tugash vaqti: %w", err)
	}
	if endExclusive.Before(start) || endExclusive.Equal(start) {
		return "", "", fmt.Errorf("tugash vaqti boshlanishdan keyin bo'lishi kerak")
	}
	return start.Format("2006-01-02 15:04:05"), endExclusive.Format("2006-01-02 15:04:05"), nil
}

func parseReportBound(raw string, asExclusiveEnd bool) (time.Time, error) {
	raw = strings.TrimSpace(strings.ReplaceAll(raw, "T", " "))
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)

	if raw == "" {
		if asExclusiveEnd {
			return today.AddDate(0, 0, 1), nil
		}
		return today, nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	var parsed time.Time
	var err error
	matchedDateOnly := false
	for _, layout := range layouts {
		parsed, err = time.ParseInLocation(layout, raw, time.Local)
		if err == nil {
			matchedDateOnly = layout == "2006-01-02"
			break
		}
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("noto'g'ri format: %s", raw)
	}

	if asExclusiveEnd {
		if matchedDateOnly {
			return parsed.AddDate(0, 0, 1), nil
		}
		// datetime-local is minute precision — include the selected minute
		return parsed.Add(time.Minute), nil
	}
	return parsed, nil
}
