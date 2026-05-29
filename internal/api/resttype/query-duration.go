package resttype

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/tekkamanendless/restfulwrapper"
)

// Duration is a duration that can be used as a query parameter and as a JSON field.
type Duration time.Duration

var _ json.Unmarshaler = new(Duration)
var _ json.Marshaler = new(Duration)
var _ restfulwrapper.ParameterParser = new(Duration)

func (v *Duration) ParseString(input string) error {
	durationValue, err := time.ParseDuration(input)
	if err != nil {
		return fmt.Errorf("could not parse duration from %q: %w", input, err)
	}

	*v = Duration(durationValue)
	return nil
}

func (v *Duration) UnmarshalJSON(input []byte) error {
	var stringValue string
	err := json.Unmarshal(input, &stringValue)
	if err != nil {
		return fmt.Errorf("could not unmarshal duration from %q: %w", input, err)
	}

	timeValue, err := time.ParseDuration(stringValue)
	if err != nil {
		return fmt.Errorf("could not parse duration from %q: %w", stringValue, err)
	}

	*v = Duration(timeValue)
	return nil
}

func (v Duration) MarshalJSON() ([]byte, error) {
	contents, err := json.Marshal(time.Duration(v).String())
	if err != nil {
		return nil, err
	}
	output := string(contents)
	output = strings.ReplaceAll(output, "m0s", "m")
	output = strings.ReplaceAll(output, "h0m", "h")

	return []byte(output), nil
}
