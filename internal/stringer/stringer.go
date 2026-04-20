package stringer

import "strings"

// Join the non-empty, non-whitespace elements in the array.
//
// If there are no non-empty, non-whitespace elements in the array, then this returns the empty string.
func Join(input []string, join string) string {
	var nonEmptyStrings []string
	for _, value := range input {
		value = strings.TrimSpace(value)
		if value != "" {
			nonEmptyStrings = append(nonEmptyStrings, value)
		}
	}

	if len(nonEmptyStrings) == 0 {
		return ""
	}
	return strings.Join(nonEmptyStrings, join)
}
