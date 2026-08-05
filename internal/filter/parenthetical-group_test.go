package filter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadParentheticalGroup(t *testing.T) {
	rows := []struct {
		description string
		input       string
		success     bool
		output      string
	}{
		{
			description: "empty",
			input:       "",
			success:     false,
		},
		{
			description: "no closing paren",
			input:       "1 + 2",
			success:     false,
		},
		{
			description: "exactly one closing paren",
			input:       "1 + 2)",
			success:     true,
			output:      "1 + 2",
		},
		{
			description: "nested group",
			input:       "1 + (2 + 3))",
			success:     true,
			output:      "1 + ( 2 + 3 )",
		},
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			tokenList, err := Tokenize(row.input)
			require.NoError(t, err)
			output, err := ReadParentheticalGroup(tokenList)
			if !row.success {
				require.Error(t, err)
				require.Nil(t, output)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, output)
			var tokenStrings []string
			for _, token := range output {
				tokenStrings = append(tokenStrings, token.String())
			}
			require.Equal(t, row.output, strings.Join(tokenStrings, " "))
		})
	}
}
