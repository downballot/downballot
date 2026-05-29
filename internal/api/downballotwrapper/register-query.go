package downballotwrapper

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/downballot/downballot/internal/reflecthelper"
	"github.com/emicklei/go-restful/v3"
	"github.com/tekkamanendless/restfulwrapper"
	"gorm.io/gorm"
)

type restfulFunctionDatabaseField struct {
	Where          string
	WhereArguments []string
	Order          []string
}

func init() {
	// database.query can be used to query the database for one or many things using a filter.
	// Item *schema.Item `api:"database.query:where:id = ?,ID;order:name`
	restfulwrapper.Register("database.query", func(apiTagValue string, field reflect.StructField, info *restfulwrapper.RestfulFunctionInfo) (restfulwrapper.InputFieldFunction, error) {
		databaseField := restfulFunctionDatabaseField{
			Where:          "",
			WhereArguments: nil,
			Order:          nil,
		}

		databaseParts := strings.Split(apiTagValue, ";")
		for _, databasePart := range databaseParts {
			databasePartParts := strings.SplitN(databasePart, ":", 2)
			databasePartKey := databasePartParts[0]
			var databasePartValue string
			if len(databasePartParts) > 1 {
				databasePartValue = databasePartParts[1]
			}

			if databasePartKey == "" {
				continue
			}

			switch databasePartKey {
			case "where":
				filterParts := strings.Split(databasePartValue, ",")
				for filterPartIndex, filterPart := range filterParts {
					filterParts[filterPartIndex] = strings.TrimSpace(filterPart)
				}

				databaseField.Where = filterParts[0]
				if databaseField.Where == "" {
					return nil, fmt.Errorf("invalid filter for field %s: empty value", field.Name)
				}

				for _, filterWhereName := range filterParts[1:] {
					_, found := reflecthelper.FindRecursiveField(info.InMetadataType, filterWhereName)
					if !found {
						return nil, fmt.Errorf("invalid filter for field %s: unknown reference field %q", field.Name, filterWhereName)
					}
					databaseField.WhereArguments = append(databaseField.WhereArguments, filterWhereName)
				}
			case "order":
				for _, item := range strings.Split(databasePartValue, ",") {
					item = strings.TrimSpace(item)
					databaseField.Order = append(databaseField.Order, item)
				}
			default:
				return nil, fmt.Errorf("unhandled database tag: %s", databasePartKey)
			}
		}

		return func(v reflect.Value, req *restful.Request, metadataValue reflect.Value) error {
			ctx := req.Request.Context()

			db := getDatabaseFromRequest(req)
			if db == nil {
				return fmt.Errorf("could not get database from request")
			}

			// This is a database object that we need to query for.
			query := db.Session(&gorm.Session{})

			filterWhereHasANilValue := false
			if databaseField.Where != "" {
				var filterWhereArguments []any

				for _, filterWhereName := range databaseField.WhereArguments {
					filterWhereValue, err := reflecthelper.FindRecursiveValue(metadataValue, filterWhereName)
					if err != nil {
						slog.InfoContext(ctx, fmt.Sprintf("The filter where parameter %q has no value: %v", field.Name, err))
						filterWhereHasANilValue = true
						break
					}
					if !filterWhereValue.CanInterface() {
						slog.InfoContext(ctx, fmt.Sprintf("The filter where parameter %q cannot interface; this filter is not possible.", field.Name))
						filterWhereHasANilValue = true
						break
					}
					if filterWhereValue.Interface() == nil {
						slog.InfoContext(ctx, fmt.Sprintf("The filter where parameter %q is nil; this filter is not possible.", field.Name))
						filterWhereHasANilValue = true
						break
					}
					filterWhereArguments = append(filterWhereArguments, filterWhereValue.Interface())
				}

				query = query.Where(databaseField.Where, filterWhereArguments...)
			}
			for order := range databaseField.Order {
				query = query.Order(order)
			}

			if filterWhereHasANilValue {
				if field.Type.Kind() == reflect.Pointer {
					slog.InfoContext(ctx, fmt.Sprintf("The filter for %s is not good, but it's a pointer, so it's okay for it to be nil.", field.Name))
					v.SetZero()
					return nil
				}
				return fmt.Errorf("invalid filter due to a nil value")
			}

			var err error
			if field.Type.Kind() == reflect.Slice {
				err = query.Find(v.Addr().Interface()).Error
			} else {
				err = query.First(v.Addr().Interface()).Error
			}
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					if field.Type.Kind() == reflect.Pointer {
						slog.InfoContext(ctx, fmt.Sprintf("Could not find record for %s, but it's a pointer, so it's okay for it to be nil.", field.Name))
						v.SetZero()
						return nil
					}
					return restfulwrapper.NewAPIResponseError(http.StatusNotFound, "Not Found")
				}
				return err
			}

			return nil
		}, nil
	})
}
