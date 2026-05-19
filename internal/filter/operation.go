package filter

// Operation constants.
const (
	OperationEquals             string = "="
	OperationGreaterThan        string = ">"
	OperationGreaterThanOrEqual string = ">="
	OperationIs                 string = "is"
	OperationIsNot              string = "is not"
	OperationLessThan           string = "<"
	OperationLessThanOrEqual    string = "<="
	OperationNotEquals          string = "!="
	OperationWildcard           string = "~"
)

// ValidOperationMap is a map of valid operations.
var ValidOperationMap = map[string]bool{
	OperationEquals:             true,
	OperationGreaterThan:        true,
	OperationGreaterThanOrEqual: true,
	OperationIs:                 true,
	OperationIsNot:              false, // This must not be specified directly.
	OperationLessThan:           true,
	OperationLessThanOrEqual:    true,
	OperationNotEquals:          true,
	OperationWildcard:           true,
}
