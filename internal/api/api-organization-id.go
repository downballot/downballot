package api

import "github.com/downballot/downballot/internal/schema"

type hasOrganization struct {
	OrganizationID string              `api:"path:organization_id" description:"The organization ID"`
	Organization   schema.Organization `api:"database.query:where:id = ?,OrganizationID"`
}
