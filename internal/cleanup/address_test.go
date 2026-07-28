package cleanup

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddress(t *testing.T) {
	rows := []struct {
		input   string
		output  string
		success bool
	}{
		{
			input:   "123 Main St, Anytown, DE 19711, USA",
			output:  "123 Main St, Anytown, DE 19711",
			success: true,
		},
		{
			input:   "Office Building, 123 Main St, Anytown, DE 19711, USA",
			output:  "123 Main St, Anytown, DE 19711",
			success: true,
		},
		{
			input:   "Building 7, 123 Main St, Anytown, DE 19711, USA",
			output:  "123 Main St, Anytown, DE 19711",
			success: true,
		},
		{
			input:   "There are, no numbers, in this, address",
			success: false,
		},
		{
			input:   "Main St, Anytown, DE 19711, USA",
			success: false,
		},
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d", rowIndex), func(t *testing.T) {
			output, err := Address(row.input)
			if !row.success {
				require.Error(t, err)
				assert.Empty(t, output)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, row.output, output)
		})
	}
}
