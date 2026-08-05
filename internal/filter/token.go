package filter

import (
	"strings"
)

// QuoteIfNecessary quotes the input if necessary; otherwise, it returns input as-is.
func QuoteIfNecessary(input string) string {
	quote := byte(0)

	var quotableCharacters []byte
	quotableCharacters = append(quotableCharacters, ' ', '\t', '\n')
	quotableCharacters = append(quotableCharacters, '(', ')', '{', '}', '[', ']', ',', ';', '#')
	quotableCharacters = append(quotableCharacters, '<', '>', '=', '~', '-', '+', '/', '*', '&', '|', '%', '^', '!')

	for _, quotableCharacter := range quotableCharacters {
		if strings.Contains(input, string(quotableCharacter)) {
			quote = '\''
			break
		}
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

// Quoted returns true if the token is quoted.
func (t Token) Quoted() bool {
	return t.Quote != ""
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
