package permissionset

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPermission(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		rows := []struct {
			description string
			input       Permission
			output      bool
		}{
			{
				description: "Trivially empty",
				input:       "",
				output:      false,
			},
			{
				description: "Trivially small",
				input:       "a",
				output:      true,
			},
			{
				description: "Large but without a separator",
				input:       "my-permission",
				output:      true,
			},
			{
				description: "Only a separator",
				input:       ":",
				output:      false,
			},
			{
				description: "Only the left half",
				input:       "a:",
				output:      false,
			},
			{
				description: "Only the right half",
				input:       ":b",
				output:      false,
			},
			{
				description: "Two components",
				input:       "a:b",
				output:      true,
			},
			{
				description: "Smallest wildcard",
				input:       "*",
				output:      true,
			},
			{
				description: "Can star star",
				input:       "**",
				output:      true,
			},
			{
				description: "Two components with wildcard",
				input:       "*:*",
				output:      true,
			},
			{
				description: "Wildcard can be at the beginning",
				input:       "a:*b",
				output:      true,
			},
			{
				description: "Wildcard can be at the end",
				input:       "a:b*",
				output:      true,
			},
			{
				description: "Wildcard can be in the middle",
				input:       "a:b*c",
				output:      true,
			},
			{
				description: "Only letters, numbers, periods, and hyphens in components",
				input:       "a:aA09.-",
				output:      true,
			},
			{
				description: "No other wildcards are valid",
				input:       "a:[a-z]?",
				output:      false,
			},
		}
		for rowIndex, row := range rows {
			t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
				output := row.input.Valid()
				require.Equal(t, row.output, output)
			})
		}
	})
	t.Run("Matches", func(t *testing.T) {
		rows := []struct {
			description string
			permission1 Permission
			permission2 Permission
			output      bool
		}{
			{
				description: "Trivially empty",
				permission1: "",
				permission2: "",
				output:      false,
			},
			{
				description: "Equal with one component",
				permission1: "test",
				permission2: "test",
				output:      true,
			},
			{
				description: "Equal with two components",
				permission1: "test:a",
				permission2: "test:a",
				output:      true,
			},
			{
				description: "Equal with three components",
				permission1: "test:a:b",
				permission2: "test:a:b",
				output:      true,
			},
			{
				description: "Equal but with invalid right",
				permission1: "test",
				permission2: "?",
				output:      false,
			},
			{
				description: "Equal but with invald left",
				permission1: "?",
				permission2: "test",
				output:      false,
			},
			{
				description: "Not equal",
				permission1: "color:red",
				permission2: "color:blue",
				output:      false,
			},
			{
				description: "Read only match with wildcard first",
				permission1: "*:read",
				permission2: "test:read",
				output:      true,
			},
			{
				description: "Read only match with wildard second",
				permission1: "test:read",
				permission2: "*:read",
				output:      true,
			},
			{
				description: "Read only not match with wildcard first",
				permission1: "*:write",
				permission2: "test:read",
				output:      false,
			},
			{
				description: "Read only not match with wildcard second",
				permission1: "test:write",
				permission2: "*:read",
				output:      false,
			},
			{
				description: "Read for any color",
				permission1: "color*:read",
				permission2: "color.red:read",
				output:      true,
			},
			{
				description: "Smaller permission implies all larger ones",
				permission1: "*",
				permission2: "a:b",
				output:      true,
			},
			{
				description: "Smaller permission implies all larger ones",
				permission1: "*",
				permission2: "a:b:c",
				output:      true,
			},
			{
				description: "Smaller permission implies all larger ones",
				permission1: "a",
				permission2: "a:b:c",
				output:      true,
			},
			{
				description: "Smaller permission implies all larger ones",
				permission1: "a:*",
				permission2: "a:b:c",
				output:      true,
			},
		}
		for rowIndex, row := range rows {
			t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
				output := row.permission1.Matches(row.permission2)
				require.Equal(t, row.output, output)
			})
		}
	})
}
