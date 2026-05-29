package iam

import "github.com/downballot/downballot/internal/permissionset"

// IAM constants
const (
	IAMOrganizationCreate     permissionset.Permission = "organization:create"
	IAMOrganizationDelete     permissionset.Permission = "organization:delete"
	IAMOrganizationRead       permissionset.Permission = "organization:read"
	IAMOrganizationUpdate     permissionset.Permission = "organization:update"
	IAMOrganizationUserCreate permissionset.Permission = "organization.user:create"
	IAMOrganizationUserDelete permissionset.Permission = "organization.user:delete"
	IAMOrganizationUserRead   permissionset.Permission = "organization.user:read"
	IAMOrganizationUserUpdate permissionset.Permission = "organization.user:update"
	IAMGroupCreate            permissionset.Permission = "group:create"
	IAMGroupDelete            permissionset.Permission = "group:delete"
	IAMGroupRead              permissionset.Permission = "group:read"
	IAMGroupUpdate            permissionset.Permission = "group:update"
	IAMFilterCreate           permissionset.Permission = "filter:create"
	IAMFilterDelete           permissionset.Permission = "filter:delete"
	IAMFilterRead             permissionset.Permission = "filter:read"
	IAMFilterUpdate           permissionset.Permission = "filter:update"
	IAMPersonCreate           permissionset.Permission = "person:create"
	IAMPersonDelete           permissionset.Permission = "person:delete"
	IAMPersonRead             permissionset.Permission = "person:read"
	IAMPersonUpdate           permissionset.Permission = "person:update"
)

// Permissions is the definitive list of all valid permissions.
//
// No wildcard permissions are valid here.
var Permissions = []permissionset.Permission{
	IAMOrganizationCreate,
	IAMOrganizationDelete,
	IAMOrganizationRead,
	IAMOrganizationUpdate,
	IAMOrganizationUserCreate,
	IAMOrganizationUserDelete,
	IAMOrganizationUserRead,
	IAMOrganizationUserUpdate,
	IAMGroupCreate,
	IAMGroupDelete,
	IAMGroupRead,
	IAMGroupUpdate,
	IAMFilterCreate,
	IAMFilterDelete,
	IAMFilterRead,
	IAMFilterUpdate,
	IAMPersonCreate,
	IAMPersonDelete,
	IAMPersonRead,
	IAMPersonUpdate,
}
