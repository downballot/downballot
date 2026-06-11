package resttype

import (
	"strings"

	"github.com/tekkamanendless/restfulwrapper"
)

// StringList is a list of strings.
type StringList []string

var _ restfulwrapper.ParameterParser = new(StringList)

func (v *StringList) ParseString(input string) error {
	parts := strings.Split(input, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	*v = (StringList)(parts)
	return nil
}
