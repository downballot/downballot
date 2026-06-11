package reflecthelper

import (
	"fmt"
	"reflect"
	"strings"
)

// FindRecursiveField finds a field in a type recursively.
// If the field is found, then the field is returned and the boolean is true.
// If the field is not found, then the empty field is returned and the boolean is false.
func FindRecursiveField(t reflect.Type, fieldPath string) (output reflect.StructField, found bool) {
	fieldNames := strings.Split(fieldPath, ".")
	if len(fieldNames) == 0 {
		return output, false
	}

	fieldName := fieldNames[0]
	remainingFieldName := ""
	if len(fieldNames) > 1 {
		remainingFieldName = strings.Join(fieldNames[1:], ".")
	}
	//fmt.Printf("FindRecursiveField: fieldName: %q (remaining: %q)\n", fieldName, remainingFieldName)

	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return output, false
	}

	field, found := t.FieldByName(fieldName)
	if !found {
		// Loop through any anonymous structs.
		for i := range t.NumField() {
			testField := t.Field(i)
			if !testField.Anonymous {
				continue
			}

			fieldViaAnonymous, foundViaAnonymous := FindRecursiveField(testField.Type, fieldPath)
			if foundViaAnonymous {
				return fieldViaAnonymous, true
			}
		}

		return output, false
	}

	if remainingFieldName == "" {
		return field, true
	}
	return FindRecursiveField(field.Type, remainingFieldName)
}

// FindRecursiveValue finds a value in a value recursively.
// If the value is found, then the value is returned and the error is nil.
// If the value is not found, then the empty value is returned and the error is not nil.
func FindRecursiveValue(value reflect.Value, fieldName string) (reflect.Value, error) {
	return findRecursiveValue(value, fieldName, "")
}

// FindRecursiveValue finds a value in a value recursively.
// If the value is found, then the value is returned and the error is nil.
// If the value is not found, then the empty value is returned and the error is not nil.
//
// fieldPathFull is the path of the field so far; it is used for error messages.
func findRecursiveValue(value reflect.Value, fieldPath string, fieldPathFull string) (reflect.Value, error) {
	// Do a quick check to see if field even makes sense to ask for.
	_, found := FindRecursiveField(value.Type(), fieldPath)
	if !found {
		return reflect.Value{}, fmt.Errorf("field %q not found", fieldPath)
	}

	var output reflect.Value

	fieldNames := strings.Split(fieldPath, ".")
	if len(fieldNames) == 0 {
		return output, fmt.Errorf("field %q could not be split", fieldPath)
	}

	fieldName := fieldNames[0]
	remainingFieldName := ""
	if len(fieldNames) > 1 {
		remainingFieldName = strings.Join(fieldNames[1:], ".")
	}
	//fmt.Printf("FindRecursiveValue: fieldName: %q (remaining: %q) (fieldPathFull: %q)\n", fieldName, remainingFieldName, fieldPathFull)

	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return output, fmt.Errorf("field %q is nil", fieldPathFull)
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return output, fmt.Errorf("field %q is not a struct: %s", fieldPathFull, value.Kind())
	}

	fieldValue := value.FieldByName(fieldName)
	if !fieldValue.IsValid() {
		// Loop through any anonymous structs.
		// Note: It's really difficult to try to figure out if a field in a value comes from an anonymous struct.
		// Note: We've already verified that this is possible via FindRecursiveField, so we should just be able to
		// Note: check all the fields in the value recursively and return when we get a hit.
		for i := range value.NumField() {
			testField := value.Field(i)
			isNil := false
			for testField.Kind() == reflect.Pointer {
				if testField.IsNil() {
					isNil = true
					break
				}
				testField = testField.Elem()
			}
			if isNil {
				continue
			}
			if !testField.IsValid() {
				continue
			}
			if testField.Kind() != reflect.Struct {
				continue
			}

			fieldValueViaAnonymous, errViaAnonymous := FindRecursiveValue(testField, fieldName)
			if errViaAnonymous == nil {
				return fieldValueViaAnonymous, nil
			}
		}

		if !fieldValue.IsValid() {
			return output, fmt.Errorf("field %q not found in value", fieldPathFull)
		}
	}

	if remainingFieldName == "" {
		return fieldValue, nil
	}

	if fieldPathFull == "" {
		fieldPathFull = fieldName
	} else {
		fieldPathFull = fmt.Sprintf("%s.%s", fieldPathFull, fieldName)
	}
	return findRecursiveValue(fieldValue, remainingFieldName, fieldPathFull)
}
