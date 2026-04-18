package api

import (
	"bytes"
	"context"
	"encoding/csv"
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

type PostOrganizationIDPersonImportMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_    string `api:"httppath:/organization/{organization_id}/person/import"`
	_    string `api:"doc" description:"Import a new set of persons."`
	_    string `api:"notes" description:"This imports a new set of persons."`
	Body []byte `api:"body:consumes:application/octet-stream"`
}

func (a *API) PostOrganizationIDPersonImport(ctx context.Context, meta PostOrganizationIDPersonImportMetadata) (output downballotapi.Envelope[downballotapi.ImportPersonResponse], err error) {
	contents := meta.Body
	slog.InfoContext(ctx, fmt.Sprintf("Contents: (%d)", len(contents)))

	csvReader := csv.NewReader(bytes.NewReader(contents))
	csvReader.Comma = '\t'
	rows, err := csvReader.ReadAll()
	if err != nil {
		return output, err
	}
	slog.InfoContext(ctx, fmt.Sprintf("Rows: (%d)", len(rows)))
	for r := range rows {
		for c := range rows[r] {
			rows[r][c] = strings.TrimSpace(rows[r][c])
		}
	}

	var header []string
	if len(rows) > 0 {
		header = rows[0]
		rows = rows[1:]
		slog.InfoContext(ctx, fmt.Sprintf("Header: (%d)", len(header)))
	}
	slog.InfoContext(ctx, fmt.Sprintf("Rows: (%d)", len(rows)))
	for rowIndex, row := range rows {
		slog.InfoContext(ctx, fmt.Sprintf("Row[%d]: (%d)", rowIndex, len(row)))
		for h, name := range header {
			slog.InfoContext(ctx, fmt.Sprintf("   %s: %s", name, row[h]))
		}
	}

	const (
		ColumnCounty                        = "county"
		ColumnNameFirst                     = "name_first"
		ColumnNameMiddle                    = "name_middle"
		ColumnNameLast                      = "name_last"
		ColumnNameSuffix                    = "name_suffix"
		ColumnPoliticalParty                = "political_party"
		ColumnResidentialAddressDevelopment = "residential_address_development"
		ColumnVoterID                       = "voter_id"
	)
	columnMap := map[string]string{
		"County":                    ColumnCounty,
		"Name-First":                ColumnNameFirst,
		"Name_First":                ColumnNameFirst,
		"Name-Middle":               ColumnNameMiddle,
		"Name_Middle":               ColumnNameMiddle,
		"Name-Last":                 ColumnNameLast,
		"Name_Last":                 ColumnNameLast,
		"Name-Suffix":               ColumnNameSuffix,
		"Name_Suffix":               ColumnNameSuffix,
		"Political Party":           ColumnPoliticalParty,
		"Political_Party":           ColumnPoliticalParty,
		"Res Addr-Development Name": ColumnResidentialAddressDevelopment,
		"Voter ID":                  ColumnVoterID,
		"Voter_ID":                  ColumnVoterID,
	}

	var persons []*schema.Person
	for _, row := range rows {
		data := map[string]string{}
		for h, name := range header {
			internalName := columnMap[name]
			if internalName != "" {
				data["::"+internalName] = row[h]
			}
			data["vf::"+name] = row[h]
		}

		// Build the address.

		// Build the mailing address.

		person := &schema.Person{
			OrganizationID: meta.Organization.ID,
		}
		person.VoterID = data["::"+ColumnVoterID]
		person.Fields = data

		persons = append(persons, person)
	}

	output.Data.Records = uint64(len(persons))
	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Session(&gorm.Session{NewDB: true}).
			Create(&persons).
			Error
		if err != nil {
			return err
		}

		var fields []*schema.PersonField
		for _, person := range persons {
			for name, value := range person.Fields {
				field := &schema.PersonField{
					PersonID: person.ID,
					Name:     name,
					Value:    value,
				}
				fields = append(fields, field)
			}
		}
		err = tx.Session(&gorm.Session{NewDB: true}).
			Create(&fields).
			Error
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return output, err
	}

	return output, nil
}

type GetOrganizationIDPersonMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_      string `api:"httppath:/organization/{organization_id}/person"`
	_      string `api:"doc" description:"List the persons."`
	_      string `api:"notes" description:"This lists the persons."`
	Filter string `api:"query:filter"`
}

func (a *API) GetOrganizationIDPerson(ctx context.Context, meta GetOrganizationIDPersonMetadata) (output downballotapi.Envelope[downballotapi.ListPersonsResponse], err error) {
	slog.InfoContext(ctx, fmt.Sprintf("Filter string: %s", meta.Filter))
	clause, err := filter.Parse(ctx, meta.Filter)
	if err != nil {
		return output, restfulwrapper.NewAPIQueryParameterError("filter", err)
	}

	var persons []*schema.Person
	query := meta.DB.Session(&gorm.Session{})
	query = query.Where("organization_id = ?", meta.Organization.ID)
	err = query.
		Find(&persons).
		Error
	if err != nil {
		return output, err
	}

	personFieldsMap := map[uint64][]*schema.PersonField{}
	{
		var fields []*schema.PersonField
		err = meta.DB.Session(&gorm.Session{}).
			Where("person_id IN (SELECT id FROM person WHERE organization_id = ?)", meta.Organization.ID).
			Find(&fields).
			Error
		if err != nil {
			return output, err
		}
		for _, field := range fields {
			personFieldsMap[field.PersonID] = append(personFieldsMap[field.PersonID], field)
		}
	}

	hierarchies, err := getGroupHierarchiesForUser(meta.DB, meta.CurrentUser.ID, meta.Organization.ID)
	if err != nil {
		return output, err
	}
	slog.InfoContext(ctx, fmt.Sprintf("Hierarchies: (%d)", len(hierarchies)))
	for hierachyIndex, hierarchy := range hierarchies {
		slog.InfoContext(ctx, fmt.Sprintf("   [%d]: (%d)", hierachyIndex, len(hierarchy)))
		for _, group := range hierarchy {
			slog.InfoContext(ctx, fmt.Sprintf("      * %s", group.Name))
		}
	}

	output.Data.Persons = []*downballotapi.Person{}
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

		// Handle the group filters (permissions).
		hierarchiesMatch := false
		for _, hierarchy := range hierarchies {
			hierarchyMatch := true

			for _, group := range hierarchy {
				groupClause, err := filter.Parse(ctx, group.Filter)
				if err != nil {
					return output, err
				}

				match, err := groupClause.Evaluate(o.Fields)
				if err != nil {
					return output, err
				}
				if !match {
					hierarchyMatch = false
					break
				}
			}

			if hierarchyMatch {
				hierarchiesMatch = true
				break
			}
		}
		if !hierarchiesMatch {
			continue
		}

		// Handle the endpoint filter.
		match, err := clause.Evaluate(o.Fields)
		if err != nil {
			return output, err
		}
		if match {
			output.Data.Persons = append(output.Data.Persons, o)
		}
	}

	return output, nil
}
