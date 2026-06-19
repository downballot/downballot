package filter

// Clause is a clause.  Different kinds of clauses should implement this interface.
type Clause interface {
	String() string
}
