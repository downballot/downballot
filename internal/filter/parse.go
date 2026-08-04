package filter

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// Parse parses a string and returns a Clause.
func Parse(ctx context.Context, input string) (Clause, error) {
	tokenList, err := Tokenize(input)
	if err != nil {
		return nil, err
	}

	//slog.DebugContext(ctx, fmt.Sprintf("Tokens: (%d)", len(tokenList.Tokens())))
	for _, token := range tokenList.Tokens() {
		slog.DebugContext(ctx, fmt.Sprintf("* %s (quoted: %t)", token.Value, token.Quote != ""))
	}

	clause, err := ParseTokens(tokenList)
	if err != nil {
		return nil, err
	}
	return clause, nil
}

func ParseTokens(tokenList *TokenList) (Clause, error) {
	output := &ClauseGroup{
		Operation: ClauseGroupOperationOr,
	}
	andGroup := &ClauseGroup{
		Operation: ClauseGroupOperationAnd,
	}

	for !tokenList.IsEmpty() {
		token, err := tokenList.Next()
		if err != nil {
			return nil, fmt.Errorf("error reading token: %w", err)
		}

		if TokenIsClosingParen(token) {
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

			token, err = tokenList.Next()
			if err != nil {
				return nil, fmt.Errorf("error reading token: %w", err)
			}
		} else if len(andGroup.Clauses) > 0 {
			if !(token.Quote == "" && strings.Compare(strings.ToLower(token.Value), "and") == 0) {
				return nil, fmt.Errorf("missing: AND before %q", token.Value)
			}
		}

		if TokenIsOpeningParen(token) {
			group, err := ReadParentheticalGroup(tokenList)
			if err != nil {
				return nil, err
			}

			//fmt.Printf("group: %+v\n", group) // DEBUG
			clause, err := ParseTokens(NewTokenList(group))
			if err != nil {
				return nil, err
			}
			andGroup.Clauses = append(andGroup.Clauses, clause)
			continue
		}

		fieldName := token.Value

		token, err = tokenList.Next()
		if err != nil {
			return nil, fmt.Errorf("missing operation")
		}
		if token.Quote != "" {
			return nil, fmt.Errorf("invalid operation: %s", token.String())
		}
		operation := strings.ToLower(token.Value)

		if !ValidOperationMap[operation] {
			return nil, fmt.Errorf("invalid operation: %s", operation)
		}

		token, err = tokenList.Next()
		if err != nil {
			return nil, fmt.Errorf("missing operation value")
		}

		var clause Clause
		switch operation {
		case OperationIs:
			switch strings.ToLower(token.Value) {
			case "null":
				clause = &ClauseIsNull{
					Name: fieldName,
				}
			case "not":
				token, err = tokenList.Next()
				if err != nil {
					return nil, fmt.Errorf("missing value for is not operation")
				}
				if !token.Quoted() && strings.ToLower(token.Value) != "null" {
					return nil, fmt.Errorf("invalid value for is not operation: %s", token.Value)
				}
				clause = &ClauseIsNotNull{
					Name: fieldName,
				}
			default:
				return nil, fmt.Errorf("invalid value for is operation: %s", token.Value)
			}
		default:
			newClause := &ClauseCondition{
				Name:      fieldName,
				Operation: operation,
			}

			if token.Quote == "" && token.Value == "(" {
				group, err := ReadParentheticalGroup(tokenList)
				if err != nil {
					return nil, err
				}

				// Ensure that the group is comma-separated.
				for groupIndex, groupToken := range group {
					fmt.Printf("groupIndex: %d, groupToken: %s\n", groupIndex, groupToken.String())
					if groupIndex%2 == 0 {
						if groupToken.Quote == "" && groupToken.Value == "," {
							return nil, fmt.Errorf("unexpected comma in parenthetical group")
						}
						newClause.Values = append(newClause.Values, groupToken.Value)
					} else {
						if groupToken.Quote != "" || groupToken.Value != "," {
							return nil, fmt.Errorf("expected comma in position %d in parenthetical group", groupIndex+1)
						}
					}
				}
			} else {
				newClause.Values = []string{token.Value}
			}

			clause = newClause
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
