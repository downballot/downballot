package filter

import (
	"context"
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

func TestParseQuery(t *testing.T) {
	ctx := context.Background()

	rows := []struct {
		description string
		query       string
		success     bool
		canonical   string
	}{
		{
			description: "Empty",
			query:       "",
			success:     true,
			canonical:   "",
		},
		{
			description: "Simple condition",
			query:       "key1 = value1",
			success:     true,
			canonical:   "key1 = value1",
		},
		{
			description: "Trivial quotes",
			query:       "'key1' = 'value1'",
			success:     true,
			canonical:   "key1 = value1",
		},
		{
			description: "Quoted condition",
			query:       "'key 1' = 'value 1'",
			success:     true,
			canonical:   "'key 1' = 'value 1'",
		},
		{
			description: "Quoted parens",
			query:       "key1 = 'value (((1'",
			success:     true,
			canonical:   "key1 = 'value (((1'",
		},
		{
			description: "Bogus operation",
			query:       "key1 * value1",
			success:     false,
		},
		{
			description: "Unterminated quote",
			query:       "key1 = 'value1",
			success:     false,
		},
		{
			description: "Multiple AND conditions",
			query:       "key1 = value1 and key2 = value2",
			success:     true,
			canonical:   "(key1 = value1 AND key2 = value2)",
		},
		{
			description: "Multiple AND conditions with extra parens",
			query:       "(((key1 = value1 and key2 = value2)))",
			success:     true,
			canonical:   "(key1 = value1 AND key2 = value2)",
		},
		{
			description: "Multiple AND conditions with quotes",
			query:       "key1 = 'value \"1\"' and 'key 2' = \"value '2'\"",
			success:     true,
			canonical:   "(key1 = 'value \"1\"' AND 'key 2' = 'value \\'2\\'')",
		},
		{
			description: "Extra leading AND",
			query:       "and key1 = value1 and key2 = value2",
			success:     false,
		},
		{
			description: "Multiple OR conditions",
			query:       "key1 = value1 or key2 = value2",
			success:     true,
			canonical:   "(key1 = value1 OR key2 = value2)",
		},
		{
			description: "Extra leading OR",
			query:       "or key1 = value1 or key2 = value2",
			success:     false,
		},
		{
			description: "AND OR grouping",
			query:       "key1 = value1 and key2 = value2 or key3 = value3",
			success:     true,
			canonical:   "((key1 = value1 AND key2 = value2) OR key3 = value3)",
		},
		{
			description: "AND OR grouping",
			query:       "key1 = value1 and key2 = value2 or key3 = value3 and key4 = value4",
			success:     true,
			canonical:   "((key1 = value1 AND key2 = value2) OR (key3 = value3 AND key4 = value4))",
		},
		{
			description: "Real world test grouping 1",
			query:       "((((district_representative = 'RD23') AND (political_party = 'DEMOCRATIC') AND ((political_party = 'DEMOCRATIC') AND (voting_history.pr2024 = yes OR voting_history.pr2022 = yes)))))",
			success:     true,
			canonical:   "(district_representative = RD23 AND political_party = DEMOCRATIC AND (political_party = DEMOCRATIC AND (voting_history.pr2024 = yes OR voting_history.pr2022 = yes)))",
		},
		{
			description: "Real world test grouping 2",
			query:       "((((district_representative = 'RD23'))) OR (((district_representative = 'RD23') AND (political_party = 'DEMOCRATIC'))))",
			success:     true,
			canonical:   "(district_representative = RD23 OR (district_representative = RD23 AND political_party = DEMOCRATIC))",
		},
		{
			description: "Mismatched open paren",
			query:       "key1 = value1 and (key2 = value2",
			success:     false,
		},
		{
			description: "Mismatched close paren",
			query:       "key1 = value1 and )key2 = value2",
			success:     false,
		},
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			output, err := Parse(ctx, row.query)
			if !row.success {
				require.NotNil(t, err, "err is nil")
				require.Nil(t, output, "output is not nil")
			} else {
				require.Nil(t, err, "err is not nil")
				require.NotNil(t, output, "output is nil")

				assert.Equal(t, row.canonical, output.String(), "canonical is incorrect")
			}
		})
	}
}
