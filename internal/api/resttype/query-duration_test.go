package resttype

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDuration(t *testing.T) {
	rows := []struct {
		description string
		input       string
		success     bool
		output      time.Duration
	}{
		{
			description: "trivial",
			input:       "",
			success:     false,
		},
		{
			description: "1h",
			input:       "1h",
			success:     true,
			output:      1 * time.Hour,
		},
		{
			description: "1h2m",
			input:       "1h2m",
			success:     true,
			output:      1*time.Hour + 2*time.Minute,
		},
		{
			description: "1h2m3s",
			input:       "1h2m3s",
			success:     true,
			output:      1*time.Hour + 2*time.Minute + 3*time.Second,
		},
		{
			description: "1h3s",
			input:       "1h3s",
			success:     true,
			output:      1*time.Hour + 3*time.Second,
		},
	}

	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			var output Duration
			err := output.ParseString(row.input)
			if !row.success {
				require.NotNil(t, err)
				return
			}
			require.Nil(t, err)
			assert.Equal(t, row.output, time.Duration(output))

			contents, err := json.Marshal(output)
			require.Nil(t, err)
			assert.Equal(t, `"`+row.input+`"`, string(contents))

			var output2 Duration
			err = json.Unmarshal(contents, &output2)
			require.Nil(t, err)
			assert.Equal(t, row.output, time.Duration(output2))
		})
	}
}
