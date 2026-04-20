package filter

func Walk(clause Clause, f func(clause Clause)) {
	switch typedClause := clause.(type) {
	case *ClauseCondition:
	case *ClauseGroup:
		for _, groupClause := range typedClause.Clauses {
			f(groupClause)
		}
	}
}
