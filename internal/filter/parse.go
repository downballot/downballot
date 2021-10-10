package filter

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
)

// Operation constants.
const (
	OperationEquals   string = "="
	OperationContains string = "~"
)

// Clause is a clause.  Different kinds of clauses should implement this interface.
type Clause interface {
	Evaluate(fields map[string]string) (bool, error)
	String() string
}

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

type Token struct {
	Value string
	Quote string
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

// Parse parses a string and returns a Clause.
func Parse(ctx context.Context, input string) (Clause, error) {
	input += "\n"

	var tokens []*Token
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
			case '"', '\'':
				if currentToken == nil {
					currentToken = &Token{
						Quote: string(input[i]),
					}
				} else {
					if currentToken.Quote == "" {
						return nil, fmt.Errorf("unexpected quote: %q", input[i])
					} else if currentToken.Quote == string(input[i]) {
						// End the current token.
						tokens = append(tokens, currentToken)

						currentToken = nil
					} else {
						currentToken.Value += string(input[i])
					}
				}
			default:
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

	logrus.WithContext(ctx).Infof("Tokens: (%d)", len(tokens))
	for _, token := range tokens {
		logrus.WithContext(ctx).Infof("* %s (quoted: %t)", token.Value, token.Quote != "")
	}

	clause, err := ParseTokens(tokens)
	if err != nil {
		return nil, err
	}
	return clause, nil
}

func ParseTokens(tokens []*Token) (Clause, error) {
	output := &ClauseOrGroup{}
	andGroup := &ClauseAndGroup{}

	for len(tokens) > 0 {
		token := tokens[0]
		tokens = tokens[1:]

		if token.Quote == "" && token.Value == ")" {
			return nil, fmt.Errorf("unexpected close paren")
		}

		if token.Quote == "" && strings.Compare(strings.ToLower(token.Value), "or") == 0 {
			if len(andGroup.Clauses) > 0 {
				output.Clauses = append(output.Clauses, andGroup)
			}
			andGroup = &ClauseAndGroup{}
			continue
		}

		if token.Quote == "" && strings.Compare(strings.ToLower(token.Value), "and") == 0 {
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

				if token.Quote == "" && token.Value == "(" {
					parens++
					continue
				}
				if token.Quote == "" && token.Value == ")" {
					parens--
					if parens == 0 {
						break
					}
					continue
				}
				group = append(group, token)
			}
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
			OperationContains: true,
			OperationEquals:   true,
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
