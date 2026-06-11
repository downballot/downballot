package permissionset

import (
	"path"
	"strings"
)

// Permission represents a permission.
//
// A permission is a series of components, separated by ":".
//
// Permissions are typically 2 components (resource and action), but could be more, depending on the context.
type Permission string

// Concrete returns true if this permission is concrete (has no wildcards).
func (p Permission) Concrete() bool {
	return !strings.Contains(string(p), "*")
}

// Components returns the components of a permission.
func (p Permission) Components() []Component {
	parts := strings.Split(string(p), ":")
	components := make([]Component, 0, len(parts))
	for _, part := range parts {
		components = append(components, Component(part))
	}
	return components
}

// Valid returns true if this permission is syntactically valid.
func (p Permission) Valid() bool {
	components := p.Components()
	if len(components) == 0 {
		return false
	}

	for _, component := range components {
		if !component.Valid() {
			return false
		}
	}

	return true
}

// Matches returns true if the permissions match.
//
// This applies the wildcard matches in both directions.
func (p Permission) Matches(p2 Permission) bool {
	if !p.Valid() || !p2.Valid() {
		return false
	}

	pComponents := p.Components()
	p2Components := p2.Components()
	for len(pComponents) < len(p2Components) {
		pComponents = append(pComponents, Component("*"))
	}
	for len(p2Components) < len(pComponents) {
		p2Components = append(p2Components, Component("*"))
	}

	pStrings := make([]string, len(pComponents))
	for i, component := range pComponents {
		pStrings[i] = string(component)
	}
	p2Strings := make([]string, len(p2Components))
	for i, component := range p2Components {
		p2Strings[i] = string(component)
	}
	pPath := strings.Join(pStrings, "/")
	p2Path := strings.Join(p2Strings, "/")
	matched, _ := path.Match(pPath, p2Path)
	if matched {
		return true
	}
	matched, _ = path.Match(p2Path, pPath)
	if matched {
		return true
	}
	return false
}

// Component represents a single component of a permission.
//
// A component may consist of letters, numbers, periods, and hyphens.
//
// In addition, a component may have any number of wildcard characters, "*".
type Component string

// Valid returns true if this component is syntactically valid.
func (c Component) Valid() bool {
	if len(c) == 0 {
		return false
	}

	for _, r := range string(c) {
		if r >= 'a' && r <= 'z' {
			continue
		}
		if r >= 'A' && r <= 'Z' {
			continue
		}
		if r >= '0' && r <= '9' {
			continue
		}
		if r == '.' {
			continue
		}
		if r == '-' {
			continue
		}
		if r == '*' {
			continue
		}
		return false
	}
	return true
}
