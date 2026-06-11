package filter

import (
	"fmt"
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
			case '<', '>', '=', '-', '+', '/', '*', '&', '|', '%', '^', '!':
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
