package resttype

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tekkamanendless/restfulwrapper"
)

// Secret is a secret value.
type Secret string

var _ json.Unmarshaler = new(Secret)
var _ json.Marshaler = new(Secret)
var _ restfulwrapper.ParameterParser = new(Secret)

func (v *Secret) ParseString(input string) error {
	*v = Secret(input)
	return nil
}

func (v *Secret) UnmarshalJSON(input []byte) error {
	var stringValue string
	err := json.Unmarshal(input, &stringValue)
	if err != nil {
		return fmt.Errorf("could not unmarshal secret value: %w", err) // Do not include the original value.
	}

	*v = Secret(stringValue)
	return nil
}

func (v Secret) MarshalJSON() ([]byte, error) {
	var prettyValue string
	{
		frontLength := 2
		backLength := 4
		minimumLength := frontLength + backLength + 10
		prettyLength := frontLength + backLength + 20

		if len(v) < minimumLength {
			prettyValue = strings.Repeat("*", prettyLength)
		} else {
			prettyValue = string(v[:frontLength]) + strings.Repeat("*", prettyLength-frontLength-backLength) + string(v[len(v)-backLength:])
		}
	}
	return json.Marshal(prettyValue)
}
