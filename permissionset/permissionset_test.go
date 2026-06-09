package permissionset

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPermissionSet(t *testing.T) {
	t.Run("Basic setup", func(t *testing.T) {
		t.Run("Empty", func(t *testing.T) {
			p := NewPermissionSet()
			assert.False(t, p.Match(""))
			assert.False(t, p.Match("permission1:read"))
			assert.Equal(t, []Permission{}, p.Permissions())
		})
		t.Run("Non-empty", func(t *testing.T) {
			p := NewPermissionSet("permission1:read", "permission2:read")
			assert.False(t, p.Match(""))
			assert.False(t, p.Match("bogus"))
			assert.False(t, p.Match("permission1:write"))
			assert.True(t, p.Match("permission1:read"))
			assert.True(t, p.Match("permission2:read"))
			assert.Equal(t, []Permission{"permission1:read", "permission2:read"}, p.Permissions())
		})
		t.Run("Can remove non-existent permission", func(t *testing.T) {
			p := NewPermissionSet()
			assert.False(t, p.Match("permission1:read"))
			assert.Equal(t, []Permission{}, p.Permissions())
			p.RemovePermission("permission1:read")
			assert.False(t, p.Match("permission1:read"))
			assert.Equal(t, []Permission{}, p.Permissions())
		})
		t.Run("Can add and remove a permission", func(t *testing.T) {
			p := NewPermissionSet()
			assert.False(t, p.Match("permission1:read"))
			assert.Equal(t, []Permission{}, p.Permissions())
			p.AddPermission("permission1:read")
			assert.Equal(t, []Permission{"permission1:read"}, p.Permissions())
			assert.True(t, p.Match("permission1:read"))
			p.RemovePermission("permission1:read")
			assert.False(t, p.Match("permission1:read"))
			assert.Equal(t, []Permission{}, p.Permissions())
		})
	})
	t.Run("Match", func(t *testing.T) {
		rows := []struct {
			description string
			permissions []Permission
			input       []Permission
			output      bool
		}{
			{
				description: "Empty set won't match a permission",
				permissions: nil,
				input:       []Permission{"permission1:read"},
				output:      false,
			},
			{
				description: "Empty set won't match a wildcard permission",
				permissions: nil,
				input:       []Permission{"*"},
				output:      false,
			},
			{
				description: "Wildcard set will match nothing",
				permissions: []Permission{"*"},
				input:       nil,
				output:      true,
			},
			{
				description: "Wildcard set will match anything",
				permissions: []Permission{"*"},
				input:       []Permission{"permission1:read"},
				output:      true,
			},
			{
				description: "Wildcard set will match any number of things",
				permissions: []Permission{"*"},
				input:       []Permission{"permission1:read", "permission2:read"},
				output:      true,
			},
			{
				description: "Example read-only account matches read permissions",
				permissions: []Permission{"*:read"},
				input:       []Permission{"permission1:read", "permission2:read"},
				output:      true,
			},
			{
				description: "Example read-only account fails with a write permission",
				permissions: []Permission{"*:read"},
				input:       []Permission{"permission1:write"},
				output:      false,
			},
			{
				description: "Example read-only account fails with a mixed read-write permission request",
				permissions: []Permission{"*:read"},
				input:       []Permission{"permission1:read", "permission2:write"},
				output:      false,
			},
			{
				description: "Example read-only account will match a specific read request",
				permissions: []Permission{"*:read"},
				input:       []Permission{"permission1:read:specific"},
				output:      true,
			},
		}
		for rowIndex, row := range rows {
			t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
				p := NewPermissionSet(row.permissions...)
				assert.Equal(t, row.output, p.MatchAll(row.input...))
			})
		}
	})
}
