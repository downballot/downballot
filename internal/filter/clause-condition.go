package filter

// ClauseCondition is a single condition.
type ClauseCondition struct {
	Name      string   // This is the field name.
	Operation string   // This is the operation.
	Values    []string // This is the list of values that could match; essentially, this an "IN" operation.
}

var _ Clause = (*ClauseCondition)(nil)

// String returns the canonical form of the clause.
func (c ClauseCondition) String() string {
	output := QuoteIfNecessary(c.Name) + " " + c.Operation + " "
	if len(c.Values) == 1 {
		output += QuoteIfNecessary(c.Values[0])
	} else {
		output += "("
		for valueIndex, value := range c.Values {
			if valueIndex > 0 {
				output += ", "
			}
			output += QuoteIfNecessary(value)
		}
		output += ")"
	}
	return output
}
