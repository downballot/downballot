package filter

import (
	"strings"
)

type ClauseAndGroup struct {
	Clauses []Clause
}

// String returns the canonical form of the clause.
func (c ClauseAndGroup) String() string {
	var parts []string
	for _, clause := range c.Clauses {
		parts = append(parts, clause.String())
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return "(" + strings.Join(parts, " AND ") + ")"
}

// Evaluate the clause against a map.
func (c ClauseAndGroup) Evaluate(fields map[string]string) (bool, error) {
	if len(c.Clauses) == 0 {
		return true, nil
	}

	for _, clause := range c.Clauses {
		result, err := clause.Evaluate(fields)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}
