package filter

import "fmt"

// ReadParentheticalGroup reads a parenthetical group from the tokens.
// This assumes that the first paren has already been read.
func ReadParentheticalGroup(tokenList *TokenList) ([]Token, error) {
	parens := 1
	var group []Token
	for !tokenList.IsEmpty() {
		token, err := tokenList.Next()
		if err != nil {
			return nil, fmt.Errorf("error reading token: %w", err)
		}

		if TokenIsOpeningParen(token) {
			parens++
			// Don't include the first paren when building out the group.
			if parens == 1 {
				continue
			}
		} else if TokenIsClosingParen(token) {
			parens--
			// Don't include the last paren when building out the group.
			if parens == 0 {
				break
			}
		}
		group = append(group, token)
	}
	if parens > 0 {
		return nil, fmt.Errorf("mismatched parens: %d", parens)
	}
	return group, nil
}
