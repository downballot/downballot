package api

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/schema"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func (i *Instance) registerPersonEndpoints(ws *restful.WebService) {
	ws.Route(
		ws.POST("person/import").To(i.importPerson).
			Doc(`Import a new set of persons`).
			Notes(`This imports a new set of persons.`).
			Do(i.doRequireAuthentication).
			Consumes("application/octet-stream").
			Returns(http.StatusOK, "OK", downballotapi.ImportPersonResponse{}),
	)
	ws.Route(
		ws.GET("person").To(i.listPersons).
			Doc(`List the persons`).
			Notes(`This lists the persons.`).
			Do(i.doRequireAuthentication).
			Returns(http.StatusOK, "OK", downballotapi.ListPersonsResponse{}),
	)
}

func (i *Instance) importPerson(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	contents, err := ioutil.ReadAll(request.Request.Body)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}
	logrus.WithContext(ctx).Infof("Contents: (%d)", len(contents))

	csvReader := csv.NewReader(bytes.NewReader(contents))
	csvReader.Comma = '\t'
	rows, err := csvReader.ReadAll()
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}
	logrus.WithContext(ctx).Infof("Rows: (%d)", len(rows))
	for r := range rows {
		for c := range rows[r] {
			rows[r][c] = strings.TrimSpace(rows[r][c])
		}
	}

	var header []string
	if len(rows) > 0 {
		header = rows[0]
		rows = rows[1:]
		logrus.WithContext(ctx).Infof("Header: (%d)", len(header))
	}
	logrus.WithContext(ctx).Infof("Rows: (%d)", len(rows))
	for rowIndex, row := range rows {
		logrus.WithContext(ctx).Infof("Row[%d]: (%d)", rowIndex, len(row))
		for h, name := range header {
			logrus.WithContext(ctx).Infof("   %s: %s", name, row[h])
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
		"Name-Middle":               ColumnNameMiddle,
		"Name-Last":                 ColumnNameLast,
		"Name-Suffix":               ColumnNameSuffix,
		"Political Party":           ColumnPoliticalParty,
		"Res Addr-Development Name": ColumnResidentialAddressDevelopment,
		"Voter ID":                  ColumnVoterID,
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

		person := &schema.Person{}
		person.VoterID = data["::"+ColumnVoterID]
		person.Fields = data

		persons = append(persons, person)
	}

	output := downballotapi.ImportPersonResponse{
		Records: uint64(len(persons)),
	}
	err = i.App.DB.Transaction(func(tx *gorm.DB) error {
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
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	WriteEntity(ctx, response, output)
}

func (i *Instance) listPersons(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	var persons []*schema.Person
	query := i.App.DB.Session(&gorm.Session{NewDB: true})
	if request.Attribute(AttributeUserID) != nil {
		query = query.Where("organization_id IN (?)", i.App.DB.Session(&gorm.Session{NewDB: true}).Table(schema.UserOrganizationMap{}.TableName()).Select("id").Where("user_id = ?", request.Attribute(AttributeUserID)))
	}
	err := query.
		Find(&persons).
		Error
	if err != nil {
		logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	output := downballotapi.ListPersonsResponse{
		Persons: []*downballotapi.Person{},
	}
	for _, person := range persons {
		o := &downballotapi.Person{
			ID:      fmt.Sprintf("%d", person.ID),
			VoterID: person.VoterID,
			Fields:  map[string]string{},
		}

		var fields []*schema.PersonField
		err = i.App.DB.Session(&gorm.Session{NewDB: true}).
			Where("person_id = ?", person.ID).
			Find(&fields).
			Error
		if err != nil {
			logrus.WithContext(ctx).Warnf("Error: [%T] %v", err, err)
			WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
			return
		}
		for _, field := range fields {
			o.Fields[field.Name] = field.Value
		}

		output.Persons = append(output.Persons, o)
	}
	WriteEntity(ctx, response, output)
}
