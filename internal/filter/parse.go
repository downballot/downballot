package filter

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Parse parses a string and returns a Clause.
func Parse(ctx context.Context, input string) (Clause, error) {
	tokens, err := Tokenize(input)
	if err != nil {
		return nil, err
	}

	//slog.DebugContext(ctx, fmt.Sprintf("Tokens: (%d)", len(tokens)))
	for _, token := range tokens {
		slog.DebugContext(ctx, fmt.Sprintf("* %s (quoted: %t)", token.Value, token.Quote != ""))
	}

	clause, err := ParseTokens(tokens)
	if err != nil {
		return nil, err
	}
	return clause, nil
}

func ParseTokens(tokens []*Token) (Clause, error) {
	output := &ClauseGroup{
		Operation: ClauseGroupOperationOr,
	}
	andGroup := &ClauseGroup{
		Operation: ClauseGroupOperationAnd,
	}

	for len(tokens) > 0 {
		token := tokens[0]
		tokens = tokens[1:]

		if token.Quote == "" && token.Value == ")" {
			return nil, fmt.Errorf("unexpected close paren")
		}

		if token.Quote == "" && strings.Compare(strings.ToLower(token.Value), "or") == 0 {
			if len(output.Clauses) == 0 && len(andGroup.Clauses) == 0 {
				return nil, fmt.Errorf("extra leading OR")
			}

			if len(andGroup.Clauses) > 0 {
				output.Clauses = append(output.Clauses, andGroup)
			}
			andGroup = &ClauseGroup{
				Operation: ClauseGroupOperationAnd,
			}
			continue
		}

		if token.Quote == "" && strings.Compare(strings.ToLower(token.Value), "and") == 0 {
			if len(andGroup.Clauses) == 0 {
				return nil, fmt.Errorf("extra leading AND")
			}

			token = tokens[0]
			tokens = tokens[1:]
		} else if len(andGroup.Clauses) > 0 {
			if !(token.Quote == "" && strings.Compare(strings.ToLower(token.Value), "and") == 0) {
				return nil, fmt.Errorf("missing: AND before %q", token.Value)
			}
		}

		if token.Quote == "" && token.Value == "(" {
			parens := 1
			var group []*Token
			for len(tokens) > 0 {
				token = tokens[0]
				tokens = tokens[1:]

				if token.Quote == "" {
					if token.Value == "(" {
						parens++
						// Don't include the first paren when building out the group.
						if parens == 1 {
							continue
						}
					} else if token.Value == ")" {
						parens--
						// Don't include the last paren when building out the group.
						if parens == 0 {
							break
						}
					}
				}
				group = append(group, token)
			}
			if parens > 0 {
				return nil, fmt.Errorf("mismatched parens: %d", parens)
			}
			//fmt.Printf("group: %+v\n", group) // DEBUG
			clause, err := ParseTokens(group)
			if err != nil {
				return nil, err
			}
			andGroup.Clauses = append(andGroup.Clauses, clause)
			continue
		}

		fieldName := token.Value

		if len(tokens) == 0 {
			return nil, fmt.Errorf("missing operation")
		}
		token = tokens[0]
		tokens = tokens[1:]
		if token.Quote != "" {
			return nil, fmt.Errorf("invalid operation: %s", token.String())
		}
		operation := strings.ToLower(token.Value)

		if !ValidOperationMap[operation] {
			return nil, fmt.Errorf("invalid operation: %s", operation)
		}

		if len(tokens) == 0 {
			return nil, fmt.Errorf("missing operation value")
		}
		token = tokens[0]
		tokens = tokens[1:]

		var clause Clause
		switch operation {
		case OperationIs:
			switch strings.ToLower(token.Value) {
			case "null":
				clause = &ClauseIsNull{
					Name: fieldName,
				}
			case "not":
				if len(tokens) == 0 {
					return nil, fmt.Errorf("missing value for is not operation")
				}
				token = tokens[0]
				tokens = tokens[1:]
				if strings.ToLower(token.Value) != "null" {
					return nil, fmt.Errorf("invalid value for is not operation: %s", token.Value)
				}
				clause = &ClauseIsNotNull{
					Name: fieldName,
				}
			default:
				return nil, fmt.Errorf("invalid value for is operation: %s", token.Value)
			}
		default:
			clause = &ClauseCondition{
				Name:      fieldName,
				Operation: operation,
				Value:     token.Value,
			}
		}

		andGroup.Clauses = append(andGroup.Clauses, clause)
	}
	if andGroup != nil && len(andGroup.Clauses) > 0 {
		output.Clauses = append(output.Clauses, andGroup)
	}

	if len(output.Clauses) == 1 {
		return output.Clauses[0], nil
	}

	return output, nil
}
