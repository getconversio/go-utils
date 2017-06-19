package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeStartOf(t *testing.T) {
	// February 3, 2016, 04:05:06
	tt := time.Date(2016, time.February, 3, 4, 5, 6, 0, time.UTC)

	assert.Equal(t, "2016-01-01T00:00:00Z", TimeStartOf(tt, "year").Format(time.RFC3339))
	assert.Equal(t, "2016-02-01T00:00:00Z", TimeStartOf(tt, "month").Format(time.RFC3339))
	assert.Equal(t, "2016-02-03T00:00:00Z", TimeStartOf(tt, "day").Format(time.RFC3339))
	assert.Equal(t, "2016-02-03T04:00:00Z", TimeStartOf(tt, "hour").Format(time.RFC3339))
	assert.Equal(t, "2016-02-03T04:05:00Z", TimeStartOf(tt, "minute").Format(time.RFC3339))
	assert.Equal(t, "2016-02-03T04:05:06Z", TimeStartOf(tt, "second").Format(time.RFC3339))

	// Everything else just returns the time itself.
	assert.Equal(t, "2016-02-03T04:05:06Z", TimeStartOf(tt, "total").Format(time.RFC3339))

	// Test all days of the week for week start.
	dates := []time.Time{
		time.Date(2016, time.February, 8, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 9, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 10, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 11, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 12, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 13, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 14, 4, 5, 6, 0, time.UTC),
	}
	for _, date := range dates {
		assert.Equal(t, "2016-02-08T00:00:00Z", TimeStartOf(date, "week").Format(time.RFC3339))
	}
}

func TestTimeEndof(t *testing.T) {
	// February 3, 2016, 04:05:06
	tt := time.Date(2016, time.February, 3, 4, 5, 6, 0, time.UTC)

	assert.Equal(t, "2016-12-31T23:59:59Z", TimeEndOf(tt, "year").Format(time.RFC3339))
	assert.Equal(t, "2016-02-29T23:59:59Z", TimeEndOf(tt, "month").Format(time.RFC3339))
	assert.Equal(t, "2016-02-03T23:59:59Z", TimeEndOf(tt, "day").Format(time.RFC3339))
	assert.Equal(t, "2016-02-03T04:59:59Z", TimeEndOf(tt, "hour").Format(time.RFC3339))
	assert.Equal(t, "2016-02-03T04:05:59Z", TimeEndOf(tt, "minute").Format(time.RFC3339))
	assert.Equal(t, "2016-02-03T04:05:06Z", TimeEndOf(tt, "second").Format(time.RFC3339))

	// Everything else just returns the time itself.
	assert.Equal(t, "2016-02-03T04:05:06Z", TimeEndOf(tt, "total").Format(time.RFC3339))

	// Edge case for month around end of month
	tt = time.Date(2016, time.March, 31, 4, 5, 6, 0, time.UTC)
	assert.Equal(t, "2016-03-31T23:59:59Z", TimeEndOf(tt, "month").Format(time.RFC3339))
	tt = time.Date(2016, time.January, 31, 4, 5, 6, 0, time.UTC)
	assert.Equal(t, "2016-01-31T23:59:59Z", TimeEndOf(tt, "month").Format(time.RFC3339))

	// Edge case around end of leap year (2016 was a leap year)
	tt = time.Date(2016, time.December, 31, 4, 5, 6, 0, time.UTC)
	assert.Equal(t, "2016-12-31T23:59:59Z", TimeEndOf(tt, "year").Format(time.RFC3339))

	// Edge case around end of year before a leap year.
	tt = time.Date(2015, time.December, 31, 4, 5, 6, 0, time.UTC)
	assert.Equal(t, "2015-12-31T23:59:59Z", TimeEndOf(tt, "year").Format(time.RFC3339))

	// Test all days of the week for week end.
	dates := []time.Time{
		time.Date(2016, time.February, 8, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 9, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 10, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 11, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 12, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 13, 4, 5, 6, 0, time.UTC),
		time.Date(2016, time.February, 14, 4, 5, 6, 0, time.UTC),
	}
	for _, date := range dates {
		assert.Equal(t, "2016-02-14T23:59:59Z", TimeEndOf(date, "week").Format(time.RFC3339))
	}
}