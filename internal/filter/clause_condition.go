package filter

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ClauseCondition is a single condition.
type ClauseCondition struct {
	Name      string
	Operation string
	Value     string
}

var _ Clause = (*ClauseCondition)(nil)

// String returns the canonical form of the clause.
func (c ClauseCondition) String() string {
	output := QuoteIfNecessary(c.Name) + " " + c.Operation + " " + QuoteIfNecessary(c.Value)
	return output
}

// Evaluate the clause against a map.
func (c ClauseCondition) Evaluate(fields map[string]string) (bool, error) {
	value := fields[c.Name]
	switch c.Operation {
	case OperationWildcard:
		ok, err := filepath.Match(strings.ToLower(c.Value), strings.ToLower(value))
		if err != nil {
			return false, fmt.Errorf("could not perform wildcard match: %w", err)
		}
		if !ok {
			return false, nil
		}
	case OperationEquals:
		if strings.Compare(strings.ToLower(value), strings.ToLower(c.Value)) != 0 {
			return false, nil
		}
	default:
		return false, fmt.Errorf("invalid operation: %s", c.Operation)
	}
	return true, nil
}
