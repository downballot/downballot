package resttype

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDateTime(t *testing.T) {
	rows := []struct {
		description string
		input       string
		success     bool
		output      time.Time
	}{
		{
			description: "trivial",
			input:       "",
			success:     false,
		},
		{
			description: "2025-01-01",
			input:       "2025-01-01",
			success:     true,
			output:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}

	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			var output DateTime
			err := output.ParseString(row.input)
			if !row.success {
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)
			assert.Equal(t, row.output, time.Time(output))

			contents, err := json.Marshal(output)
			require.Nil(t, err)
			assert.Equal(t, `"`+row.input+`"`, string(contents))

			var output2 DateTime
			err = json.Unmarshal(contents, &output2)
			require.Nil(t, err)
			assert.Equal(t, row.output, time.Time(output2))
		})
	}
}
