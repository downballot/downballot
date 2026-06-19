package filter

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
