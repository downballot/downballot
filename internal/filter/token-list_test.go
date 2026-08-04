package filter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTokenList(t *testing.T) {
	var tokenList TokenList
	tokenList.Add(Token{Value: "1"})
	tokenList.Add(Token{Value: "2"})
	tokenList.Add(Token{Value: "3"})

	require.Equal(t, tokenList.Tokens(), []Token{
		{Value: "1"},
		{Value: "2"},
		{Value: "3"},
	})
}
