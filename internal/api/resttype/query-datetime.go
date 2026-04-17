package resttype

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/threatmate/restfulwrapper"
)

// DateTime is a timestamp that can be used as a query parameter and as a JSON field.
type DateTime time.Time

var _ json.Unmarshaler = &DateTime{}
var _ json.Marshaler = &DateTime{}
var _ restfulwrapper.ParameterParser = &DateTime{}

var DateTimeLayouts = []string{
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02",
}

func (v *DateTime) ParseString(input string) error {
	// Parse it as a unix timestamp.
	{
		integerValue, err := strconv.ParseInt(input, 10, 64)
		if err == nil {
			*v = DateTime(time.Unix(integerValue, 0))
			return nil
		}
	}

	// Parse it as a time.Time.
	for _, layout := range DateTimeLayouts {
		timeValue, err := time.Parse(layout, input)
		if err == nil {
			*v = DateTime(timeValue)
			return nil
		}
	}

	return fmt.Errorf("could not parse datetime from %q", input)
}

func (v *DateTime) UnmarshalJSON(input []byte) error {
	// Handle an integer.
	{
		// Parse it as a unix timestamp.
		{
			var integerValue int64
			err := json.Unmarshal(input, &integerValue)
			if err == nil {
				*v = DateTime(time.Unix(integerValue, 0))
				return nil
			}
		}
	}

	// Handle a string.
	{
		var stringValue string
		err := json.Unmarshal(input, &stringValue)
		if err == nil {
			// Parse it as a unix timestamp.
			{
				integerValue, err := strconv.ParseInt(stringValue, 10, 64)
				if err == nil {
					*v = DateTime(time.Unix(integerValue, 0))
					return nil
				}
			}

			// Parse it as a time.Time.
			for _, layout := range DateTimeLayouts {
				timeValue, err := time.Parse(layout, stringValue)
				if err == nil {
					*v = DateTime(timeValue)
					return nil
				}
			}
		}
	}

	return fmt.Errorf("could not parse datetime from %q", input)
}

func (v DateTime) MarshalJSON() ([]byte, error) {
	if time.Time(v).UTC().Nanosecond() > 0 {
		return json.Marshal(time.Time(v).UTC().Format("2006-01-02T15:04:05.999999999Z07:00"))
	}
	if time.Time(v).UTC().Hour() > 0 || time.Time(v).UTC().Minute() > 0 || time.Time(v).UTC().Second() > 0 {
		return json.Marshal(time.Time(v).UTC().Format("2006-01-02T15:04:05Z07:00"))
	}
	return json.Marshal(time.Time(v).UTC().Format("2006-01-02"))
}
