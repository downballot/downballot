package stringer

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJoin(t *testing.T) {
	rows := []struct {
		description string
		input       []string
		join        string
		output      string
	}{
		{
			description: "Trivial",
			input:       nil,
			join:        "",
			output:      "",
		},
		{
			description: "Nil input, string join",
			input:       nil,
			join:        ", ",
			output:      "",
		},
		{
			description: "Empty input, string join",
			input:       []string{},
			join:        ", ",
			output:      "",
		},
		{
			description: "One item, empty join",
			input:       []string{"item1"},
			join:        "",
			output:      "item1",
		},
		{
			description: "Two items, empty join",
			input:       []string{"item1", "item2"},
			join:        "",
			output:      "item1item2",
		},
		{
			description: "Two items, comma join",
			input:       []string{"item1", "item2"},
			join:        ", ",
			output:      "item1, item2",
		},
		{
			description: "Two good items, two bad items, comma join",
			input:       []string{"item1", " ", "item2", ""},
			join:        ", ",
			output:      "item1, item2",
		},
		{
			description: "Whitespace is trimmed",
			input:       []string{"   item1 ", " ", "\t\n item2 \n\t", ""},
			join:        ", ",
			output:      "item1, item2",
		},
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			output := Join(row.input, row.join)
			require.Equal(t, row.output, output)
		})
	}
}
