package filter

// ClauseIsNotNull is a condition that checks if a field is not null.
type ClauseIsNotNull struct {
	Name string
}

var _ Clause = (*ClauseIsNotNull)(nil)

// String returns the canonical form of the clause.
func (c ClauseIsNotNull) String() string {
	output := QuoteIfNecessary(c.Name) + " IS NOT NULL"
	return output
}
