package durationparser

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDurationParser(t *testing.T) {
	t.Run("Bogus values", func(t *testing.T) {
		rows := []struct {
			description string
			input       string
		}{
			{"Empty string", ""},
			{"All text", "bogus"},
			{"Just a number", "1"},
			{"Just a number", "12"},
			{"Just a number", "123"},
			{"Just a number", "-1"},
			{"Bad unit", "1z"},
			{"Bad number", "1sx"},
		}
		for rowIndex, row := range rows {
			t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
				baseDate := time.Now()
				result, err := Parse(baseDate, row.input)
				require.NotNil(t, err)
				assert.Nil(t, result)
			})
		}
	})
	t.Run("Forever values", func(t *testing.T) {
		rows := []struct {
			description string
			input       string
		}{
			{"Forever", "forever"},
		}
		for rowIndex, row := range rows {
			t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
				baseDate := time.Now()
				result, err := Parse(baseDate, row.input)
				require.Nil(t, err)
				assert.Nil(t, result)
			})
		}
	})
	t.Run("Good values", func(t *testing.T) {
		// Note: Daylight savings time in 2014 was March 9th.
		// We'll need to test around that date when it comes to testing the "month"
		// duration.
		timezone, err := time.LoadLocation("America/New_York")
		require.Nil(t, err)
		format := "2006-01-02 15:04:05"
		date2a, err := time.ParseInLocation(format, "2014-02-05 00:00:00", timezone)
		require.Nil(t, err)
		date2b, err := time.ParseInLocation(format, "2014-02-11 00:00:00", timezone)
		require.Nil(t, err)

		rows := []struct {
			description string
			date        time.Time
			input       string
			duration    int64
		}{
			{"Seconds", date2a, "0s", 0},
			{"Seconds", date2a, "1s", 1},
			{"Seconds", date2a, "10s", 10},
			{"Seconds", date2a, "-1s", -1},
			{"Seconds", date2a, "-10s", -10},
			{"Minutes", date2a, "0m", 0},
			{"Minutes", date2a, "1m", 1 * 60},
			{"Minutes", date2a, "10m", 10 * 60},
			{"Minutes", date2a, "-1m", -1 * 60},
			{"Minutes", date2a, "-10m", -10 * 60},
			{"Hours", date2a, "0h", 0},
			{"Hours", date2a, "1h", 1 * 3600},
			{"Hours", date2a, "10h", 10 * 3600},
			{"Hours", date2a, "-1h", -1 * 3600},
			{"Hours", date2a, "-10h", -10 * 3600},
			{"Days", date2a, "0d", 0},
			{"Days", date2a, "1d", 1 * 86400},
			{"Days", date2a, "10d", 10 * 86400},
			{"Days", date2a, "-1d", -1 * 86400},
			{"Days", date2a, "-10d", -10 * 86400},
			{"Months", date2a, "0M", 0},
			{"Months", date2a, "1M", 1 * 28 * 86400},    // February 5 -> March 5 (28 days)
			{"Months", date2b, "1M", 1*28*86400 - 3600}, // February 11 -> March 11 (28 days - Daylight savings time)
			{"Months", date2a, "-1M", -1 * 31 * 86400},  // February 5 - January 5 (31 days)
			{"Years", date2a, "0y", 0},
			{"Years", date2a, "1y", 1 * 365 * 86400},
			{"Years", date2a, "-1y", -1 * 365 * 86400},
		}
		for rowIndex, row := range rows {
			t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
				result, err := Parse(row.date, row.input)
				require.Nil(t, err)
				if assert.NotNil(t, result) {
					difference := result.Unix() - row.date.Unix()
					assert.Equal(t, row.duration, difference)
				}
			})
		}
	})
}
