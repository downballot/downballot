package filter

// ClauseIsNull is a condition that checks if a field is null.
type ClauseIsNull struct {
	Name string
}

var _ Clause = (*ClauseIsNull)(nil)

// String returns the canonical form of the clause.
func (c ClauseIsNull) String() string {
	output := QuoteIfNecessary(c.Name) + " IS NULL"
	return output
}
