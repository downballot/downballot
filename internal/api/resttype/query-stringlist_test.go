package resttype

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStringList(t *testing.T) {
	rows := []struct {
		description string
		input       string
		output      []string
	}{
		{
			description: "trivial",
			input:       "",
			output:      []string{""},
		},
		{
			description: "Simple",
			input:       "1",
			output:      []string{"1"},
		},
		{
			description: "Multiple",
			input:       "1,two,THREE",
			output:      []string{"1", "two", "THREE"},
		},
		{
			description: "Whitespace ignored",
			input:       "1 , \ttwo\n,THREE",
			output:      []string{"1", "two", "THREE"},
		},
	}

	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			var output StringList
			err := output.ParseString(row.input)
			require.Nil(t, err)
			assert.Equal(t, row.output, []string(output))
		})
	}
}
