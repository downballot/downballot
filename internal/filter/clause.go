package filter

// Clause is a clause.  Different kinds of clauses should implement this interface.
type Clause interface {
	Evaluate(fields map[string]string) (bool, error)
	String() string
}
