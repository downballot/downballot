package cleanup

import (
	"fmt"
	"regexp"
	"strings"
)

var startsWithNumberRegex = regexp.MustCompile(`^[0-9]+`)

// Address cleans up an address string.
//
// This removes the USA suffix, if any, and it purges any place names up until actual street number part.
func Address(address string) (string, error) {
	address = strings.TrimSuffix(address, ", USA")

	lines := strings.Split(address, ",")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}

	for len(lines) > 0 {
		if !startsWithNumberRegex.MatchString(lines[0]) {
			lines = lines[1:]
			continue
		}

		break
	}
	if len(lines) == 0 {
		return "", fmt.Errorf("no lines starting with a number found in address: %s", address)
	}

	return strings.Join(lines, ", "), nil
}
