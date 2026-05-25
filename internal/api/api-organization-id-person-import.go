package api

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/api/restcsv"
	"github.com/downballot/downballot/internal/schema"
	"github.com/downballot/downballot/internal/schema/sqltype"
	"github.com/downballot/downballot/internal/stringer"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type PostOrganizationIDPersonImportMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.RequireAuthenticatedUser
	downballotwrapper.UseDatabase
	hasOrganization
	_        string        `api:"httppath:/organization/{organization_id}/person/import"`
	_        string        `api:"doc" description:"Import a new set of persons."`
	_        string        `api:"notes" description:"This imports a new set of persons."`
	FieldMap string        `api:"query:field_map" description:"A comma-separated list of field mappings. The format is 'source_field:destination_field'."`
	Body     restcsv.Table `api:"body:consumes:text/csv"`
}

func (a *API) PostOrganizationIDPersonImport(ctx context.Context, meta PostOrganizationIDPersonImportMetadata) (output downballotapi.Envelope[downballotapi.ImportPersonResponse], err error) {
	slog.InfoContext(ctx, fmt.Sprintf("Header: %+v", meta.Body.Header))
	slog.InfoContext(ctx, fmt.Sprintf("Rows: (%d)", len(meta.Body.Rows)))

	var fieldDefinitions []*schema.PersonFieldDefinition
	err = meta.DB.Session(&gorm.Session{}).
		Where("organization_id = ?", meta.Organization.ID).
		Find(&fieldDefinitions).
		Error
	if err != nil {
		return output, fmt.Errorf("could not find field definitions: %w", err)
	}

	fieldDefinitionByNameMap := map[string]*schema.PersonFieldDefinition{}
	fieldDefinitionByIDMap := map[uint64]*schema.PersonFieldDefinition{}
	for _, fieldDefinition := range fieldDefinitions {
		fieldDefinitionByNameMap[fieldDefinition.Name] = fieldDefinition
		fieldDefinitionByIDMap[fieldDefinition.ID] = fieldDefinition
	}

	// Parse the field map.
	fieldMap := map[string]string{}
	for _, mapping := range strings.Split(meta.FieldMap, ",") {
		mapping = strings.TrimSpace(mapping)
		if mapping == "" {
			continue
		}
		parts := strings.SplitN(mapping, ":", 2)
		if len(parts) != 2 {
			return output, restfulwrapper.NewAPIQueryParameterError("field_map", fmt.Errorf("invalid mapping: %q", mapping))
		}
		csvName := strings.TrimSpace(parts[0])
		internalName := strings.TrimSpace(parts[1])
		if _, ok := fieldDefinitionByNameMap[internalName]; !ok {
			return output, fmt.Errorf("unknown field: %q", internalName)
		}
		fieldMap[csvName] = internalName
	}

	// Trim all of the cells.
	for c := range meta.Body.Header {
		meta.Body.Header[c] = strings.TrimSpace(meta.Body.Header[c])
	}
	for r := range meta.Body.Rows {
		for c := range meta.Body.Rows[r] {
			meta.Body.Rows[r][c] = strings.TrimSpace(meta.Body.Rows[r][c])
		}
	}

	// Tansform the header:
	// * All lowercase
	// * Averything except letters, numbers, and "." replaced with "_"
	// * Trim "_" from the ends
	// * Merge all contiguous "_"
	{
		illegalCharacters, err := regexp.Compile(`[^a-z0-9.]+`)
		if err != nil {
			return output, fmt.Errorf("could not compile expression: %w", err)
		}

		for c, name := range meta.Body.Header {
			name = strings.ToLower(name)
			name = illegalCharacters.ReplaceAllString(name, "_")
			name = strings.Trim(name, "_")
			meta.Body.Header[c] = name
		}
	}

	for rowIndex, row := range meta.Body.Rows {
		slog.DebugContext(ctx, fmt.Sprintf("Row[%d]: (%d)", rowIndex, len(row)))
		for h, name := range meta.Body.Header {
			slog.DebugContext(ctx, fmt.Sprintf("   %s: %s", name, row[h]))
		}
	}

	const (
		ColumnBirthdayYear                  = "birthday_year"
		ColumnCoordinates                   = "coordinates"
		ColumnCounty                        = "county"
		ColumnDistrictRepresentative        = "district_representative"
		ColumnDistrictSchool                = "district_school"
		ColumnDistrictSenate                = "district_senate"
		ColumnName                          = "name"
		ColumnNameFirst                     = "name_first"
		ColumnNameMiddle                    = "name_middle"
		ColumnNameLast                      = "name_last"
		ColumnNameSuffix                    = "name_suffix"
		ColumnPhoneNumber                   = "phone_number"
		ColumnPoliticalParty                = "political_party"
		ColumnResidentialAddress            = "residential_address"
		ColumnResidentialAddressDevelopment = "residential_address_development"
		ColumnMailingAddress                = "mailing_address"
		ColumnVoterID                       = "voter_id"
		ColumnVotingHistory                 = "voting_history"
	)
	columnMap := map[string]string{
		"year_of_birth":             ColumnBirthdayYear,
		"county":                    ColumnCounty,
		"district_representative":   ColumnDistrictRepresentative,
		"district_school":           ColumnDistrictSchool,
		"district_senate":           ColumnDistrictSenate,
		"name_first":                ColumnNameFirst,
		"name_middle":               ColumnNameMiddle,
		"name_last":                 ColumnNameLast,
		"name_suffix":               ColumnNameSuffix,
		"political_party":           ColumnPoliticalParty,
		"res_addr_development_name": ColumnResidentialAddressDevelopment,
		"voter_id":                  ColumnVoterID,
	}
	for csvName, internalName := range fieldMap {
		columnMap[csvName] = internalName
	}

	// Add all of the field definitions to the column map.
	for _, fieldDefinition := range fieldDefinitions {
		columnMap[fieldDefinition.Name] = fieldDefinition.Name
	}

	var persons []*schema.Person
	for _, row := range meta.Body.Rows {
		// data is a structured version of the row.
		data := map[string]string{}
		for h, name := range meta.Body.Header {
			data[name] = row[h]
		}

		fields := map[string]string{}
		for name, value := range data {
			internalName := columnMap[name]
			if internalName != "" {
				fields[internalName] = value
			}
		}

		// Build the full name.
		{
			name := stringer.Join([]string{data["name_first"], data["name_middle"], data["name_last"]}, " ")
			if name != "" {
				if value := data["name_suffix"]; value != "" {
					name += ", " + value
				}
				fields[ColumnName] = name
			}
		}

		// Build the address.
		{
			address := stringer.Join([]string{
				stringer.Join([]string{
					data["res_addr_house_no"],
					data["res_addr_house_no_suffix"],
					data["res_addr_street_direction_prefix"],
					data["res_addr_street_name"],
					data["res_addr_street_type"],
					data["res_addr_street_direction_suffix"],
					data["res_addr_unit_type"],
					data["res_addr_unit_number"],
				}, " "),
				stringer.Join([]string{
					stringer.Join([]string{
						stringer.Join([]string{data["res_addr_city"]}, ""),
						stringer.Join([]string{data["res_addr_state"]}, ""),
					}, ", "),
					stringer.Join([]string{data["res_addr_zip_code"], data["res_addr_zip_4"]}, "-"),
				}, " "),
			}, ", ")
			if address != "" {
				fields[ColumnResidentialAddress] = address
			}
		}

		// Build the mailing address.
		{
			address := stringer.Join([]string{
				stringer.Join([]string{data["mail_addr_line1"]}, ""),
				stringer.Join([]string{data["mail_addr_line2"]}, ""),
				stringer.Join([]string{data["mail_addr_line3"]}, ""),
				stringer.Join([]string{data["mail_addr_line4"]}, ""),
				stringer.Join([]string{
					stringer.Join([]string{
						stringer.Join([]string{data["mail_addr_city"]}, ""),
						stringer.Join([]string{data["mail_addr_state"]}, ""),
					}, ", "),
					stringer.Join([]string{data["mail_addr_zip_code"], data["mail_addr_zip_4"]}, "-"),
				}, " "),
			}, ", ")
			if address != "" {
				fields[ColumnMailingAddress] = address
			}
		}

		// Build the phone number.
		{
			phoneNumber := stringer.Join([]string{
				data["phone_area_code"],
				data["phone_exchange"],
				data["phone_last_four"],
			}, "-")
			if phoneNumber != "" {
				fields[ColumnPhoneNumber] = phoneNumber
			}
		}

		// Build the voting history.
		{
			votes := []string{}
			votingHistoryFields := []string{}
			for name, value := range data {
				if strings.HasPrefix(name, "voting_history_") {
					votingHistoryFields = append(votingHistoryFields, name)
					vote := strings.ToLower(value)
					if vote != "" {
						votes = append(votes, vote)
					}
				}
			}
			slices.Sort(votes)

			if len(votingHistoryFields) > 0 {
				finalValue := ""
				if len(votes) > 0 {
					finalValue = "," + strings.Join(votes, ",") + "," // We want to bracket everything with commas for easier searches later.
				}
				fields[ColumnVotingHistory] = finalValue
			}
		}

		person := &schema.Person{
			OrganizationID: meta.Organization.ID,
		}
		person.VoterID = fields[ColumnVoterID]
		person.Fields = fields

		persons = append(persons, person)
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Records = uint64(len(persons))

	existingPersons := []*schema.Person{}
	voterIDToExistingPersonMap := map[string]*schema.Person{}
	err = meta.DB.Session(&gorm.Session{}).
		Where("organization_id = ?", meta.Organization.ID).
		Find(&existingPersons).
		Error
	if err != nil {
		return output, fmt.Errorf("could not find existing persons: %w", err)
	}
	for _, person := range existingPersons {
		voterIDToExistingPersonMap[person.VoterID] = person
	}

	newPersons := []*schema.Person{}
	updatePersons := []*schema.Person{}
	for _, person := range persons {
		if existingPerson, ok := voterIDToExistingPersonMap[person.VoterID]; ok {
			person.ID = existingPerson.ID
			for name, value := range person.Fields {
				fieldDefinition := fieldDefinitionByNameMap[name]
				if fieldDefinition == nil {
					return output, fmt.Errorf("unknown field: %q", name)
				}

				// This is a CSV import, and we're updating an existing person, so if the field is blank, then ignore it.
				if value == "" {
					continue
				}

				err = fieldDefinition.Validate(value)
				if err != nil {
					return output, fmt.Errorf("invalid value for field %s: %w", name, err)
				}
			}

			updatePersons = append(updatePersons, person)
		} else {
			for name, value := range person.Fields {
				fieldDefinition := fieldDefinitionByNameMap[name]
				if fieldDefinition == nil {
					return output, fmt.Errorf("unknown field: %q", name)
				}

				err = fieldDefinition.Validate(value)
				if err != nil {
					return output, fmt.Errorf("invalid value for field %s: %w", name, err)
				}
			}

			newPersons = append(newPersons, person)
		}
	}
	slog.InfoContext(ctx, fmt.Sprintf("New persons: (%d)", len(newPersons)))
	slog.InfoContext(ctx, fmt.Sprintf("Update persons: (%d)", len(updatePersons)))

	err = meta.DB.Transaction(func(tx *gorm.DB) error {
		if len(newPersons) > 0 {
			err := tx.Session(&gorm.Session{NewDB: true}).
				CreateInBatches(&newPersons, 2000).
				Error
			if err != nil {
				return err
			}

			var fields []*schema.PersonField
			for _, person := range newPersons {
				for name, value := range person.Fields {
					personFieldDefinition := fieldDefinitionByNameMap[name]
					if personFieldDefinition == nil {
						continue
					}
					field := &schema.PersonField{
						PersonID:                person.ID,
						PersonFieldDefinitionID: personFieldDefinition.ID,
						Value:                   value,
					}
					fields = append(fields, field)
				}
			}
			err = tx.Session(&gorm.Session{NewDB: true}).
				CreateInBatches(&fields, 2000).
				Error
			if err != nil {
				return err
			}
		}

		for _, updatePerson := range updatePersons {
			person := *updatePerson
			person.Fields = map[string]string{}
			{
				fields := []*schema.PersonField{}
				err = tx.Session(&gorm.Session{NewDB: true}).
					Where("person_id = ?", person.ID).
					Find(&fields).
					Error
				if err != nil {
					return err
				}
				for _, field := range fields {
					fieldDefinition := fieldDefinitionByIDMap[field.PersonFieldDefinitionID]
					if fieldDefinition == nil {
						continue
					}
					person.Fields[fieldDefinition.Name] = field.Value
				}
			}

			slog.InfoContext(ctx, fmt.Sprintf("Person %d: (%d) %+v", person.ID, len(person.Fields), person.Fields))
			slog.InfoContext(ctx, fmt.Sprintf("Update person %d: (%d) %+v", person.ID, len(updatePerson.Fields), updatePerson.Fields))

			// Do not update fields that are blank.
			//
			// Basically, we're accepting spreadsheet input, so if a field is blank, then ignore it.
			updateFields := map[string]*string{}
			for field, value := range updatePerson.Fields {
				if value != "" {
					updateFields[field] = &value
				}
			}
			slog.InfoContext(ctx, fmt.Sprintf("Update fields for person %d: (%d) %+v", person.ID, len(updateFields), updateFields))

			personID := person.ID

			for field, value := range updateFields {
				fieldDefinition := fieldDefinitionByNameMap[field]
				if fieldDefinition == nil {
					// We should have already defended against this, but play it safe.
					return fmt.Errorf("unknown field: %s", field)
				}

				audit := schema.PersonAudit{
					UserID:                  meta.CurrentUser.ID,
					PersonID:                personID,
					PersonFieldDefinitionID: fieldDefinition.ID,
					Timestamp:               sqltype.DateTime(time.Now()),
				}

				// If the field had a value, then record its old value.
				if oldValue, ok := person.Fields[field]; ok {
					audit.OldValue = new(string)
					*audit.OldValue = oldValue
				}
				// If the field has a new value, then record its new value.
				if value != nil {
					audit.NewValue = value
				}

				if audit.OldValue == nil && audit.NewValue == nil {
					// If the field was added and deleted, then don't do anything.
					continue
				}
				if audit.OldValue != nil && audit.NewValue != nil && *audit.OldValue == *audit.NewValue {
					// If the field was not changed, then don't do anything.
					continue
				}

				if value == nil {
					err := tx.Session(&gorm.Session{}).
						Where("person_id = ?", personID).
						Where("person_field_definition_id = ?", fieldDefinition.ID).
						Delete(&schema.PersonField{}).
						Error
					if err != nil {
						return fmt.Errorf("could not delete field: %w", err)
					}
				} else {
					var fields []*schema.PersonField
					err := tx.Session(&gorm.Session{}).
						Where("person_id = ?", personID).
						Where("person_field_definition_id = ?", fieldDefinition.ID).
						Find(&fields).
						Error
					if err != nil {
						return fmt.Errorf("could not find fields: %w", err)
					}

					if len(fields) == 0 {
						field := schema.PersonField{
							PersonID:                personID,
							PersonFieldDefinitionID: fieldDefinition.ID,
							Value:                   *value,
						}
						err := tx.Session(&gorm.Session{}).
							Create(&field).
							Error
						if err != nil {
							return fmt.Errorf("could not create field: %w", err)
						}
					} else {
						field := fields[0]
						err := tx.Session(&gorm.Session{}).
							Model(&schema.PersonField{}).
							Where("id = ?", field.ID).
							Update("value", *value).
							Error
						if err != nil {
							return fmt.Errorf("could not update field: %w", err)
						}
					}
				}

				err := tx.Session(&gorm.Session{}).
					Create(&audit).
					Error
				if err != nil {
					return fmt.Errorf("could not create audit: %w", err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return output, err
	}

	return output, nil
}
