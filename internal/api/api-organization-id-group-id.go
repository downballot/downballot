package api

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/filter"
	"github.com/downballot/downballot/internal/schema"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type hasGroup struct {
	GroupID string       `api:"path:group_id" description:"The group ID"`
	Group   schema.Group `api:"database.query:where:id = ? AND organization_id IN (SELECT id FROM organization),GroupID"`
}

type GetOrganizationIDGroupIDMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasGroup
	_ string `api:"httppath:/organization/{organization_id}/group/{group_id}"`
	_ string `api:"doc" description:"Get the group."`
	_ string `api:"notes" description:"This gets the group."`
}

func (a *API) GetOrganizationIDGroupID(ctx context.Context, meta GetOrganizationIDGroupIDMetadata) (output downballotapi.Envelope[downballotapi.GetGroupResponse], err error) {
	o := &downballotapi.Group{
		ID:   fmt.Sprintf("%d", meta.Group.ID),
		Name: meta.Group.Name,
	}
	if meta.Group.ParentID != nil {
		o.ParentID = fmt.Sprintf("%d", *meta.Group.ParentID)
	}
	output.Data.Group = o
	return output, nil
}

type GetOrganizationIDGroupIDPersonMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	hasGroup
	_      string  `api:"httppath:/organization/{organization_id}/group/{group_id}/person"`
	_      string  `api:"doc" description:"Get the people in the group."`
	_      string  `api:"notes" description:"This gets the people in the group."`
	Filter *string `api:"query:filter"`
	Fields *string `api:"query:fields"`
}

func (a *API) GetOrganizationIDGroupIDPerson(ctx context.Context, meta GetOrganizationIDGroupIDPersonMetadata) (output downballotapi.Envelope[downballotapi.ListPersonsResponse], err error) {
	hierarchies, err := getGroupHierarchiesForUser(meta.DB, meta.CurrentUser.ID, meta.Organization.ID)
	if err != nil {
		return output, err
	}
	slog.InfoContext(ctx, fmt.Sprintf("Hierarchies: (%d)", len(hierarchies)))

	var groupHierarchy []*schema.Group
	for _, hierarchy := range hierarchies {
		if len(hierarchy) > 0 && hierarchy[len(hierarchy)-1].ID == meta.Group.ID {
			groupHierarchy = hierarchy
			break
		}
	}

	if len(groupHierarchy) == 0 {
		return output, fmt.Errorf("could not find hierarchy for group.id=%d", meta.Group.ID)
	}

	query := meta.DB.Session(&gorm.Session{})

	var joinCount uint
	var f func(clause filter.Clause, groupQuery *gorm.DB)
	f = func(clause filter.Clause, groupQuery *gorm.DB) {
		slog.DebugContext(ctx, fmt.Sprintf("f: clause: %+v", clause))
		switch typedClause := clause.(type) {
		case *filter.ClauseCondition:
			slog.DebugContext(ctx, fmt.Sprintf("f: condition: %+v", typedClause))

			joinCount++
			query = query.Joins("LEFT OUTER JOIN person_field AS person_field_join" + fmt.Sprintf("%d", joinCount) + " ON person.id = person_field_join" + fmt.Sprintf("%d", joinCount) + ".person_id")
			groupQuery = groupQuery.Where("person_field_join"+fmt.Sprintf("%d", joinCount)+".name = ?", typedClause.Name)
			switch typedClause.Operation {
			case filter.OperationEquals:
				groupQuery = groupQuery.Where("person_field_join"+fmt.Sprintf("%d", joinCount)+".value = ?", typedClause.Value)
			case filter.OperationWildcard:
				groupQuery = groupQuery.Where("person_field_join"+fmt.Sprintf("%d", joinCount)+".value LIKE ?", strings.ReplaceAll(typedClause.Value, "*", "%"))
			default:
				slog.WarnContext(ctx, fmt.Sprintf("Unknown operation: %s", typedClause.Operation))
			}
		case *filter.ClauseGroup:
			slog.DebugContext(ctx, fmt.Sprintf("f: group: %+v", typedClause))

			for _, groupClause := range typedClause.Clauses {
				switch typedClause.Operation {
				case filter.ClauseGroupOperationAnd:
					newQuery := meta.DB.Session(&gorm.Session{NewDB: true, Initialized: true})
					f(groupClause, newQuery)
					groupQuery.Where(newQuery)
				case filter.ClauseGroupOperationOr:
					newQuery := meta.DB.Session(&gorm.Session{NewDB: true, Initialized: true})
					f(groupClause, newQuery)
					groupQuery.Or(newQuery)
				}
			}
		default:
			slog.WarnContext(ctx, fmt.Sprintf("Unknown clause type: %T", typedClause))
		}
	}

	for _, group := range groupHierarchy {
		slog.DebugContext(ctx, fmt.Sprintf("Group: id=%d, name=%s", group.ID, group.Name))

		groupClause, err := filter.Parse(ctx, group.Filter)
		if err != nil {
			return output, err
		}
		newQuery := meta.DB.Session(&gorm.Session{NewDB: true, Initialized: true})
		f(groupClause, newQuery)
		query = query.Where(newQuery)
	}

	if meta.Filter != nil {
		groupClause, err := filter.Parse(ctx, *meta.Filter)
		if err != nil {
			return output, err
		}
		newQuery := meta.DB.Session(&gorm.Session{NewDB: true, Initialized: true})
		f(groupClause, newQuery)
		query = query.Where(newQuery)
	}

	var persons []*schema.Person
	err = query.Find(&persons).
		Limit(1000). // TODO: Setting for this?
		Error
	if err != nil {
		return output, err
	}

	var personIDs []uint64
	for _, person := range persons {
		personIDs = append(personIDs, person.ID)
	}

	personFieldsMap := map[uint64][]*schema.PersonField{}
	{
		var fields []*schema.PersonField
		query := meta.DB.Session(&gorm.Session{}).
			Where("person_id IN (?)", personIDs)
		if meta.Fields != nil {
			query = query.Where("name IN (?)", strings.Split(*meta.Fields, ","))
		}
		err := query.
			Find(&fields).
			Error
		if err != nil {
			return output, err
		}
		for _, field := range fields {
			if personFieldsMap[field.PersonID] == nil {
				personFieldsMap[field.PersonID] = []*schema.PersonField{}
			}
			personFieldsMap[field.PersonID] = append(personFieldsMap[field.PersonID], field)
		}
	}

	for _, person := range persons {
		o := &downballotapi.Person{
			ID:      fmt.Sprintf("%d", person.ID),
			VoterID: person.VoterID,
			Fields:  map[string]string{},
		}

		fields := personFieldsMap[person.ID]
		for _, field := range fields {
			o.Fields[field.Name] = field.Value
		}

		output.Data.Persons = append(output.Data.Persons, o)
	}

	return output, nil
}

type GetOrganizationIDGroupRootMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_     string       `api:"httppath:/organization/{organization_id}/group/root"`
	_     string       `api:"doc" description:"Get the group."`
	_     string       `api:"notes" description:"This gets the group."`
	Group schema.Group `api:"database.query:where:parent_id IS NULL AND organization_id IN (SELECT id FROM organization)"`
}

func (a *API) GetOrganizationIDGroupRoot(ctx context.Context, meta GetOrganizationIDGroupRootMetadata) (output downballotapi.Envelope[downballotapi.GetGroupResponse], err error) {
	o := &downballotapi.Group{
		ID:   fmt.Sprintf("%d", meta.Group.ID),
		Name: meta.Group.Name,
	}
	if meta.Group.ParentID != nil {
		o.ParentID = fmt.Sprintf("%d", *meta.Group.ParentID)
	}
	output.Data.Group = o
	return output, nil
}
