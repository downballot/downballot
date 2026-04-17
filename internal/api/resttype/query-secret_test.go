package resttype

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecret(t *testing.T) {
	rows := []struct {
		description string
		input       string
		output      string
	}{
		{
			description: "trivial",
			output:      "**************************",
		},
		{
			description: "Short 1",
			input:       "1",
			output:      "**************************",
		},
		{
			description: "Short 6",
			input:       "123456",
			output:      "**************************",
		},
		{
			description: "Short 7",
			input:       "1234567",
			output:      "**************************",
		},
		{
			description: "Short 15",
			input:       "123456789012345",
			output:      "**************************",
		},
		{
			description: "Long 16",
			input:       "1234567890123456",
			output:      "12********************3456",
		},
		{
			description: "Long 17",
			input:       "12345678901234567",
			output:      "12********************4567",
		},
		{
			description: "Long 40",
			input:       "1234567890123456789012345678901234567890",
			output:      "12********************7890",
		},
	}

	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			var output Secret
			err := output.ParseString(row.input)
			require.Nil(t, err)
			assert.Equal(t, row.input, string(output))

			contents, err := json.Marshal(output)
			require.Nil(t, err)
			assert.True(t, strings.HasPrefix(string(contents), `"`))

			var output2 Secret
			err = json.Unmarshal(contents, &output2)
			require.Nil(t, err)
			assert.Equal(t, row.output, string(output2))
		})
	}
}
