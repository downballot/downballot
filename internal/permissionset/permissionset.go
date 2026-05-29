package permissionset

import "slices"

// PermissionSet is a set of permissons.
type PermissionSet struct {
	permissionMap map[Permission]struct{} // This is the set of permissions.  `struct{}` takes up no space.
}

// NewPermissionSet returns a new PermissionSet with the given permissions.
func NewPermissionSet(permissions ...Permission) *PermissionSet {
	p := &PermissionSet{}
	for _, permission := range permissions {
		p.AddPermission(permission)
	}

	return p
}

// AddPermission adds a permission to the set.
func (p *PermissionSet) AddPermission(permissions ...Permission) {
	if p.permissionMap == nil {
		p.permissionMap = map[Permission]struct{}{}
	}
	for _, permission := range permissions {
		p.permissionMap[permission] = struct{}{}
	}
}

// RemovePermission removes a permission from the set.
func (p *PermissionSet) RemovePermission(permissions ...Permission) {
	if p.permissionMap == nil {
		p.permissionMap = map[Permission]struct{}{}
	}
	for _, permission := range permissions {
		delete(p.permissionMap, permission)
	}
}

// Permissions returns the list of permissions in the set.
//
// Note that this list is a copy; modifying it has no consequence.
func (p *PermissionSet) Permissions() []Permission {
	output := []Permission{}
	for permission := range p.permissionMap {
		output = append(output, permission)
	}

	slices.Sort(output)
	return output
}

// Match returns true if the permission set includes the given permission.
//
// This will return true as long as at least one permission in the set matches
// the given permission.
func (p *PermissionSet) Match(permission Permission) bool {
	for testPerpermission := range p.permissionMap {
		if testPerpermission.Matches(permission) {
			return true
		}
	}
	return false
}

// MatchAll returns true if the permission set includes all of the permissions given.
func (p *PermissionSet) MatchAll(permissions ...Permission) bool {
	for _, permission := range permissions {
		if !p.Match(permission) {
			return false
		}
	}
	return true
}

// Intersect returns a new permission set that is the intersection of the two given permission sets.
func (p *PermissionSet) Intersect(other *PermissionSet) *PermissionSet {
	output := NewPermissionSet()
	for permission := range p.permissionMap {
		if _, ok := other.permissionMap[permission]; ok {
			output.AddPermission(permission)
		}
	}
	return output
}
