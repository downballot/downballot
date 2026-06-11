package filter

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTokenize(t *testing.T) {
	rows := []struct {
		description string
		input       string
		success     bool
		tokens      []string
	}{
		{
			description: "empty",
			input:       "",
			success:     true,
			tokens:      []string{},
		},
		{
			description: "spaces",
			input:       "   ",
			success:     true,
			tokens:      []string{},
		},
		{
			description: "empty lines",
			input:       "\n\n\n",
			success:     true,
			tokens:      []string{},
		},
		{
			description: "basic text",
			input:       "hello",
			success:     true,
			tokens:      []string{"hello"},
		},
		{
			description: "multiple text",
			input:       "hello world",
			success:     true,
			tokens:      []string{"hello", "world"},
		},
		{
			description: "multiple text with commas",
			input:       "hello, world",
			success:     true,
			tokens:      []string{"hello", ",", "world"},
		},
		{
			description: "commas",
			input:       "hello,,,world",
			success:     true,
			tokens:      []string{"hello", ",", ",", ",", "world"},
		},
		{
			description: "semicolons",
			input:       "hello;;;world",
			success:     true,
			tokens:      []string{"hello", ";", ";", ";", "world"},
		},
		{
			description: "open parens",
			input:       "hello(((world",
			success:     true,
			tokens:      []string{"hello", "(", "(", "(", "world"},
		},
		{
			description: "close parens",
			input:       "hello)))world",
			success:     true,
			tokens:      []string{"hello", ")", ")", ")", "world"},
		},
		{
			description: "open squigglies",
			input:       "hello{{{world",
			success:     true,
			tokens:      []string{"hello", "{", "{", "{", "world"},
		},
		{
			description: "close squigglies",
			input:       "hello}}}world",
			success:     true,
			tokens:      []string{"hello", "}", "}", "}", "world"},
		},
		{
			description: "open brackets",
			input:       "hello[[[world",
			success:     true,
			tokens:      []string{"hello", "[", "[", "[", "world"},
		},
		{
			description: "close brackets",
			input:       "hello]]]world",
			success:     true,
			tokens:      []string{"hello", "]", "]", "]", "world"},
		},
		{
			description: "pluses",
			input:       "hello+++world",
			success:     true,
			tokens:      []string{"hello", "+++", "world"},
		},
		{
			description: "minuses",
			input:       "hello---world",
			success:     true,
			tokens:      []string{"hello", "---", "world"},
		},
		{
			description: "stars",
			input:       "hello***world",
			success:     true,
			tokens:      []string{"hello", "***", "world"},
		},
		{
			description: "slashes",
			input:       "hello///world",
			success:     true,
			tokens:      []string{"hello", "///", "world"},
		},
		{
			description: "math",
			input:       "x=((a+b)/2*10)^2",
			success:     true,
			tokens:      []string{"x", "=", "(", "(", "a", "+", "b", ")", "/", "2", "*", "10", ")", "^", "2"},
		},
		{
			description: "plus five",
			input:       "+5",
			success:     true,
			tokens:      []string{"+", "5"},
		},
		{
			description: "minus five",
			input:       "-5",
			success:     true,
			tokens:      []string{"-", "5"},
		},
		{
			description: "quoted special tokens",
			input:       `'::vf-a+b_c' = 'test "1"'`,
			success:     true,
			tokens:      []string{`::vf-a+b_c`, `=`, `test "1"`},
		},
		{
			description: "addition",
			input:       "1 + 2",
			success:     true,
			tokens:      []string{"1", "+", "2"},
		},
		{
			description: "subtraction",
			input:       "1 - 2",
			success:     true,
			tokens:      []string{"1", "-", "2"},
		},
		{
			description: "multiplication",
			input:       "1 * 2",
			success:     true,
			tokens:      []string{"1", "*", "2"},
		},
		{
			description: "division",
			input:       "1 / 2",
			success:     true,
			tokens:      []string{"1", "/", "2"},
		},
		{
			description: "equal",
			input:       "1 = 2",
			success:     true,
			tokens:      []string{"1", "=", "2"},
		},
		{
			description: "double equal",
			input:       "1 == 2",
			success:     true,
			tokens:      []string{"1", "==", "2"},
		},
		{
			description: "not equal",
			input:       "1 != 2",
			success:     true,
			tokens:      []string{"1", "!=", "2"},
		},
		{
			description: "greater than",
			input:       "1 > 2",
			success:     true,
			tokens:      []string{"1", ">", "2"},
		},
		{
			description: "greater than or equal to",
			input:       "1 >= 2",
			success:     true,
			tokens:      []string{"1", ">=", "2"},
		},
		{
			description: "less than",
			input:       "1 < 2",
			success:     true,
			tokens:      []string{"1", "<", "2"},
		},
		{
			description: "less than or equal to",
			input:       "1 <= 2",
			success:     true,
			tokens:      []string{"1", "<=", "2"},
		},
		{
			description: "wildcard",
			input:       "1 ~ 2",
			success:     true,
			tokens:      []string{"1", "~", "2"},
		},
		{
			description: "incomplete single quote",
			input:       "hello, 'world",
			success:     false,
		},
		{
			description: "incomplete double quote",
			input:       "hello, \"world",
			success:     false,
		},
		{
			description: "is null",
			input:       "1 is null",
			success:     true,
			tokens:      []string{"1", "is", "null"},
		},
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			tokens, err := Tokenize(row.input)
			if !row.success {
				require.NotNil(t, err)
				require.Nil(t, tokens)
			} else {
				require.Nil(t, err)
				require.NotNil(t, tokens)

				tokenStrings := []string{}
				for _, token := range tokens {
					tokenStrings = append(tokenStrings, token.Value)
				}
				assert.Equal(t, row.tokens, tokenStrings)
			}
		})
	}
}
