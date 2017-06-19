package util

import "time"

// Epoch is the unix epoch time: Jan 1, 1970
var Epoch = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)

// TimeStartOf truncates the given timestamp to the given granularity. This is
// is similar to Time.Truncate, but is also works for year, month and day.
func TimeStartOf(t time.Time, granularity string) time.Time {
	switch granularity {
	case "year":
		return time.Date(t.Year(), time.January, 1, 0, 0, 0, 0, t.Location())
	case "month":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	case "week":
		// A week starts on a Monday golang!
		dayDiff := -(int(t.Weekday()) - 1) // Monday - 1 = 0
		if t.Weekday() == 0 {
			dayDiff = -6
		}
		t = t.AddDate(0, 0, dayDiff)
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "day":
		return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	case "hour":
		return t.Truncate(time.Hour)
	case "minute":
		return t.Truncate(time.Minute)
	case "second":
		return t.Truncate(time.Second)
	}
	return t
}

// TimeEndOf truncates a time to the end of a granularity, similar to TimeStartOf
func TimeEndOf(t time.Time, granularity string) time.Time {
	switch granularity {
	case "year":
		return TimeStartOf(t.AddDate(1, 0, 0), "year").Add(-1 * time.Microsecond)
	case "month":
		// Because month are normalized, adding a month to March 31 produces
		// May 1. So to avoid this, we first go back to the beginning of the
		// month before adding the new month.
		startOfMonth := TimeStartOf(t, "month")
		return TimeStartOf(startOfMonth.AddDate(0, 1, 0), "month").Add(-1 * time.Microsecond)
	case "week":
		return TimeStartOf(t.AddDate(0, 0, 7), "week").Add(-1 * time.Microsecond)
	case "day":
		return TimeStartOf(t.AddDate(0, 0, 1), "day").Add(-1 * time.Microsecond)
	case "hour":
		return TimeStartOf(t.Add(time.Hour), "hour").Add(-1 * time.Microsecond)
	case "minute":
		return TimeStartOf(t.Add(time.Minute), "minute").Add(-1 * time.Microsecond)
	case "second":
		return TimeStartOf(t.Add(time.Second), "second").Add(-1 * time.Microsecond)
	}
	return t
}
