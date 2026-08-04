package filter

import (
	"strings"
)

type ClauseGroupOperation int

const (
	ClauseGroupOperationOr ClauseGroupOperation = iota
	ClauseGroupOperationAnd
)

type ClauseGroup struct {
	Operation ClauseGroupOperation
	Clauses   []Clause
}

var _ Clause = (*ClauseGroup)(nil)

// String returns the canonical form of the clause.
func (c ClauseGroup) String() string {
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
	var operationName string
	switch c.Operation {
	case ClauseGroupOperationAnd:
		operationName = "AND"
	case ClauseGroupOperationOr:
		operationName = "OR"
	}
	return "(" + strings.Join(parts, " "+operationName+" ") + ")"
}
