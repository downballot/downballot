package normalize

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPreProcessAddress(t *testing.T) {
	rows := []struct {
		description string
		input       string
		output      string
	}{
		{
			description: "No change",
			input:       "123 main st, newark, de 19711",
			output:      "123 main st, newark, de 19711",
		},
		{
			description: "Fix christina mill",
			input:       "100 christina mill dr apt 200, newark, de 19711",
			output:      "200 christina mill dr, newark, de 19711",
		},
		{
			description: "Fix christina mill (after a previous oops)",
			input:       "100 christina mill dr #200, newark, de 19711",
			output:      "200 christina mill dr, newark, de 19711",
		},
		{
			description: "Fix room number",
			input:       "123 main st rm 101, newark, de 19711",
			output:      "123 main st apt 101, newark, de 19711",
		},
	}

	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d", rowIndex), func(t *testing.T) {
			output := PreProcessAddress(row.input)
			assert.Equal(t, row.output, output)
		})
	}
}

func TestPostProcessAddress(t *testing.T) {
	rows := []struct {
		description string
		input       string
		output      string
	}{
		{
			description: "Fix single-letter apartment number",
			input:       "123 main st a, newark, de 19711",
			output:      "123 main st #a, newark, de 19711",
		},
		{
			description: "Do not mess with east, west, north, or south",
			input:       "123 main st e, newark, de 19711",
			output:      "123 main st e, newark, de 19711",
		},
		{
			description: "Fix clear apartment number",
			input:       "123 main st a4, newark, de 19711",
			output:      "123 main st #a4, newark, de 19711",
		},
	}

	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d", rowIndex), func(t *testing.T) {
			output := PostProcessAddress(row.input)
			assert.Equal(t, row.output, output)
		})
	}
}
