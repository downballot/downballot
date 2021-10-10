package filter

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			description: "Multiple AND conditions with quotes",
			query:       "key1 = 'value \"1\"' and 'key 2' = \"value '2'\"",
			success:     true,
			canonical:   "(key1 = 'value \"1\"' AND 'key 2' = 'value \\'2\\'')",
		},
		{
			description: "Multiple OR conditions",
			query:       "key1 = value1 or key2 = value2",
			success:     true,
			canonical:   "(key1 = value1 OR key2 = value2)",
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
