package filter

import "fmt"

// TokenList is a list of tokens.
// This can be used to read chunks of tokens.
type TokenList struct {
	tokens []Token
}

// NewTokenList creates a new token list from a slice of tokens.
func NewTokenList(tokens []Token) *TokenList {
	return &TokenList{
		tokens: tokens,
	}
}

// IsEmpty returns true if the token list is empty.
func (t *TokenList) IsEmpty() bool {
	return len(t.tokens) == 0
}

// Add a token to the list.
func (t *TokenList) Add(token Token) {
	t.tokens = append(t.tokens, token)
}

// Tokens returns the list of tokens.
// This is a copy of the internal list, so it is safe to modify.
func (t *TokenList) Tokens() []Token {
	output := make([]Token, len(t.tokens))
	copy(output, t.tokens)
	return output
}

// Next returns the next token from the list.
func (t *TokenList) Next() (Token, error) {
	tokens, err := t.ReadN(1)
	if err != nil {
		return Token{}, err
	}
	return tokens[0], nil
}

// Peek returns the next token from the list without removing it.
func (t *TokenList) Peek() (Token, error) {
	tokens, err := t.PeekN(1)
	if err != nil {
		return Token{}, err
	}
	return tokens[0], nil
}

// PeekN returns the next n tokens from the list without removing them.
func (t *TokenList) PeekN(n int) ([]Token, error) {
	if len(t.tokens) < n {
		return nil, fmt.Errorf("not enough tokens")
	}
	output := make([]Token, n)
	copy(output, t.tokens[:n])
	return output, nil
}

// ReadN reads the next n tokens from the list.
func (t *TokenList) ReadN(n int) ([]Token, error) {
	if len(t.tokens) < n {
		return nil, fmt.Errorf("not enough tokens")
	}
	output := make([]Token, n)
	copy(output, t.tokens[:n])
	t.tokens = t.tokens[n:]
	return output, nil
}

// TokenIsOpeningParen returns true if the token is an opening parenthesis.
func TokenIsOpeningParen(token Token) bool {
	return !token.Quoted() && token.Value == "("
}

// TokenIsClosingParen returns true if the token is a closing parenthesis.
func TokenIsClosingParen(token Token) bool {
	return !token.Quoted() && token.Value == ")"
}

// ReadUntil reads tokens until the predicate returns true.
func (t *TokenList) ReadUntil(predicate func(Token) bool) ([]Token, error) {
	for i := range len(t.tokens) {
		if predicate(t.tokens[i]) {
			return t.ReadN(i)
		}
	}
	return nil, fmt.Errorf("predicate never returned true")
}
