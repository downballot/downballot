package downballotwrapper

type RequirePermissionFilterCreate struct {
	_ string `api:"downballot.permission:filter:create"`
}

type RequirePermissionFilterDelete struct {
	_ string `api:"downballot.permission:filter:delete"`
}

type RequirePermissionFilterRead struct {
	_ string `api:"downballot.permission:filter:read"`
}

type RequirePermissionFilterUpdate struct {
	_ string `api:"downballot.permission:filter:update"`
}

type RequirePermissionOrganizationUserCreate struct {
	_ string `api:"downballot.permission:organization.user:create"`
}

type RequirePermissionOrganizationUserDelete struct {
	_ string `api:"downballot.permission:organization.user:delete"`
}

type RequirePermissionOrganizationUserRead struct {
	_ string `api:"downballot.permission:organization.user:read"`
}

type RequirePermissionOrganizationUserUpdate struct {
	_ string `api:"downballot.permission:organization.user:update"`
}
