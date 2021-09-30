package durationparser

import (
	"fmt"
	"strconv"
	"time"
)

// Parse parses the duration string and applies to the given date.
//
// If the duration string is "forever", then this will return a nil time and no error.
func Parse(baseDate time.Time, durationString string) (*time.Time, error) {
	if durationString == "" {
		return nil, fmt.Errorf("empty duration string")
	}
	if durationString == "forever" {
		return nil, nil
	}

	if len(durationString) < 2 {
		return nil, fmt.Errorf("duration string must be at least 2 characters")
	}
	numberPart := durationString[:len(durationString)-1]
	unitPart := durationString[len(durationString)-1:]

	number, err := strconv.ParseInt(numberPart, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse integer from %q: %v", numberPart, err)
	}

	var newDate time.Time
	switch unitPart {
	case "s":
		newDate = baseDate.Add(time.Duration(number) * time.Second)
	case "m":
		newDate = baseDate.Add(time.Duration(number) * time.Minute)
	case "h":
		newDate = baseDate.Add(time.Duration(number) * time.Hour)
	case "d":
		newDate = baseDate.AddDate(0, 0, int(number))
	case "M":
		newDate = baseDate.AddDate(0, int(number), 0)
	case "y":
		newDate = baseDate.AddDate(int(number), 0, 0)
	default:
		return nil, fmt.Errorf("invalid unit: %s", unitPart)
	}
	return &newDate, nil
}
