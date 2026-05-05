package filter

// Operation constants.
const (
	OperationEquals             string = "="
	OperationNotEquals          string = "!="
	OperationGreaterThan        string = ">"
	OperationGreaterThanOrEqual string = ">="
	OperationLessThan           string = "<"
	OperationLessThanOrEqual    string = "<="
	OperationWildcard           string = "~"
)

// Clause is a clause.  Different kinds of clauses should implement this interface.
type Clause interface {
	Evaluate(fields map[string]string) (bool, error)
	String() string
}
