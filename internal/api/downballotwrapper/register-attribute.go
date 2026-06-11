package downballotwrapper

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/downballot/downballot/internal/reflecthelper"
	"github.com/emicklei/go-restful/v3"
	"github.com/tekkamanendless/restfulwrapper"
)

func init() {
	restfulwrapper.Register("attribute", func(apiTagValue string, field reflect.StructField, info *restfulwrapper.RestfulFunctionInfo) (restfulwrapper.InputFieldFunction, error) {
		parts := strings.SplitN(apiTagValue, ":", 2)
		attributeName := strings.TrimSpace(parts[0])
		attributeValueField := ""
		if len(parts) > 1 {
			attributeValueField = strings.TrimSpace(parts[1])
		}

		if attributeName == "" {
			return nil, fmt.Errorf("attribute name is required")
		}

		if attributeValueField != "" {
			_, found := reflecthelper.FindRecursiveField(info.InMetadataType, attributeValueField)
			if !found {
				return nil, fmt.Errorf("attribute value field %q not found", attributeValueField)
			}
		}

		return func(v reflect.Value, req *restful.Request, metadataValue reflect.Value) error {
			ctx := req.Request.Context()

			if attributeValueField != "" {
				slog.DebugContext(ctx, fmt.Sprintf("Setting attribute %q", attributeName))

				attributeValue, err := reflecthelper.FindRecursiveValue(metadataValue, attributeValueField)
				if err != nil {
					return fmt.Errorf("could not find attribute value field %q: %w", attributeValueField, err)
				}

				if !attributeValue.CanInterface() {
					return fmt.Errorf("attribute value field %q cannot interface", attributeValueField)
				}

				req.SetAttribute(attributeName, attributeValue.Interface())
			}

			if v.CanSet() {
				slog.DebugContext(ctx, fmt.Sprintf("Getting attribute %q", attributeName))

				attributeInterface := req.Attribute(attributeName)
				if attributeInterface == nil {
					if v.Kind() == reflect.Pointer {
						v.SetZero()
						return nil
					}
					return fmt.Errorf("attribute %q is nil", attributeName)
				}

				attributeValue := reflect.ValueOf(attributeInterface)
				if v.Type() != attributeValue.Type() {
					return fmt.Errorf("attribute %q has incorrect type %T: expected %T", attributeName, attributeValue.Type(), v.Type())
				}
				v.Set(attributeValue)
			}

			return nil
		}, nil
	})
}
