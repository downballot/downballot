package filter

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// QuoteIfNecessary quotes the input if necessary; otherwise, it returns input as-is.
func QuoteIfNecessary(input string) string {
	quote := byte(0)
	if strings.Contains(input, "'") {
		quote = '\''
	} else if strings.Contains(input, "\"") {
		quote = '\''
	} else if strings.Contains(input, " ") || strings.Contains(input, "\t") || strings.Contains(input, "\n") {
		quote = '\''
	}
	if quote == byte(0) {
		return input
	}

	var output string
	output += string(quote)
	for _, b := range input {
		if b == '\'' {
			output += "\\'"
		} else {
			output += string(b)
		}
	}
	output += string(quote)
	return output
}

// Token is a token during parsing.
type Token struct {
	Value  string // The current value of the token.  We will append to this during parsing.
	Quote  string // If this token is quoted, then this is the quote character.
	Symbol bool   // If this token is a symbol, then this is true.
}

func (t Token) String() string {
	output := t.Quote
	if t.Quote == "" {
		output += t.Value
	} else {
		output += strings.ReplaceAll(t.Value, t.Quote, "\\"+t.Quote)
	}
	output += t.Quote
	return output
}

func Tokenize(input string) ([]*Token, error) {
	input += "\n"

	tokens := []*Token{}
	{
		var currentToken *Token
		for i := 0; i < len(input); i++ {
			switch input[i] {
			case ' ', '\t', '\n':
				if currentToken == nil {
					// Extra whitespace will just be eaten up.
				} else {
					if currentToken.Quote == "" {
						// End the current token.
						tokens = append(tokens, currentToken)

						currentToken = nil
					} else {
						currentToken.Value += string(input[i])
					}
				}
			case '(', ')', '{', '}', '[', ']', ',', ';', '#':
				if currentToken == nil {
					currentToken = &Token{
						Value: string(input[i]),
					}
					tokens = append(tokens, currentToken)

					currentToken = nil
				} else {
					if currentToken.Quote == "" {
						// End the current token.
						tokens = append(tokens, currentToken)

						currentToken = &Token{
							Value: string(input[i]),
						}
						tokens = append(tokens, currentToken)

						currentToken = nil
					} else {
						currentToken.Value += string(input[i])
					}
				}
			case '<', '>', '=', '-', '+', '/', '*', '&', '|', '%', '^':
				if currentToken == nil {
					currentToken = &Token{
						Value:  string(input[i]),
						Symbol: true,
					}
				} else {
					if currentToken.Quote == "" && !currentToken.Symbol {
						// End the current token.
						tokens = append(tokens, currentToken)

						currentToken = &Token{
							Value:  string(input[i]),
							Symbol: true,
						}
					} else {
						currentToken.Value += string(input[i])
					}
				}
			case '"', '\'':
				if currentToken == nil {
					currentToken = &Token{
						Quote: string(input[i]),
					}
				} else {
					if currentToken.Quote == "" {
						return nil, fmt.Errorf("unexpected quote at [%d]: %q", i, input[i])
					} else if currentToken.Quote == string(input[i]) {
						// End the current token.
						tokens = append(tokens, currentToken)

						currentToken = nil
					} else {
						currentToken.Value += string(input[i])
					}
				}
			default:
				// TODO: This could be wrong...
				if currentToken != nil && currentToken.Symbol {
					// End the current token.
					tokens = append(tokens, currentToken)

					currentToken = nil
				}
				if currentToken == nil {
					currentToken = &Token{}
				}
				currentToken.Value += string(input[i])
			}
		}
		if currentToken != nil {
			return nil, fmt.Errorf("incomlete token: %q", currentToken.Value)
		}
	}

	return tokens, nil
}

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

		clause := &ClauseCondition{
			Name: token.Value,
		}

		if len(tokens) == 0 {
			return nil, fmt.Errorf("missing operation")
		}
		token = tokens[0]
		tokens = tokens[1:]
		if token.Quote != "" {
			return nil, fmt.Errorf("invalid operation: %s", token.String())
		}
		clause.Operation = token.Value

		validOperationMap := map[string]bool{
			OperationWildcard:    true,
			OperationEquals:      true,
			OperationGreaterThan: true,
			OperationLessThan:    true,
		}
		if !validOperationMap[clause.Operation] {
			return nil, fmt.Errorf("invalid operation: %s", clause.Operation)
		}

		if len(tokens) == 0 {
			return nil, fmt.Errorf("missing value")
		}
		token = tokens[0]
		tokens = tokens[1:]
		clause.Value = token.Value

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
