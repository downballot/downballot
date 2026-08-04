package filter

import "fmt"

func Tokenize(input string) (*TokenList, error) {
	input += "\n"

	tokens := []Token{}
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
						tokens = append(tokens, *currentToken)

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
					tokens = append(tokens, *currentToken)

					currentToken = nil
				} else {
					if currentToken.Quote == "" {
						// End the current token.
						tokens = append(tokens, *currentToken)

						currentToken = &Token{
							Value: string(input[i]),
						}
						tokens = append(tokens, *currentToken)

						currentToken = nil
					} else {
						currentToken.Value += string(input[i])
					}
				}
			case '<', '>', '=', '~', '-', '+', '/', '*', '&', '|', '%', '^', '!':
				if currentToken == nil {
					currentToken = &Token{
						Value:  string(input[i]),
						Symbol: true,
					}
				} else {
					if currentToken.Quote == "" && !currentToken.Symbol {
						// End the current token.
						tokens = append(tokens, *currentToken)

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
						tokens = append(tokens, *currentToken)

						currentToken = nil
					} else {
						currentToken.Value += string(input[i])
					}
				}
			default:
				// TODO: This could be wrong...
				if currentToken != nil && currentToken.Symbol {
					// End the current token.
					tokens = append(tokens, *currentToken)

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

	tokenList := TokenList{
		tokens: tokens,
	}
	return &tokenList, nil
}
