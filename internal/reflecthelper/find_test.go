package reflecthelper

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFind(t *testing.T) {
	type Subsubstruct struct {
		Subsubfield1 string
	}
	type Substruct struct {
		Subsubstruct // Ensure that we have an anonymous struct field.
		Field1       string
		Subfield1    string
	}
	type TestStruct struct {
		Substruct // Ensure that we have an anonymous struct field.
		Field1    string
		Field2    string
		Field3    struct {
			Field1 string
			Field2 string
		}
		Field4 *struct {
			Field1 string
			Field2 string
		}
	}

	t.Run("Will not work for non-struct types", func(t *testing.T) {
		t.Run("int", func(t *testing.T) {
			value := 1
			t.Run("Field", func(t *testing.T) {
				field, found := FindRecursiveField(reflect.TypeOf(value), "Field1")
				require.False(t, found)
				require.Empty(t, field.Name)
			})
			t.Run("Value", func(t *testing.T) {
				value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1")
				require.Error(t, err)
				assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field1"))
				require.Empty(t, value)
			})
		})
		t.Run("string", func(t *testing.T) {
			value := "some-string"
			t.Run("Field", func(t *testing.T) {
				field, found := FindRecursiveField(reflect.TypeOf(value), "Field1")
				require.False(t, found)
				require.Empty(t, field.Name)
			})
			t.Run("Value", func(t *testing.T) {
				value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1")
				require.Error(t, err)
				assert.Contains(t, err.Error(), "Field1")
				require.Empty(t, value)
			})
		})
		t.Run("float64", func(t *testing.T) {
			value := 1.0
			t.Run("Field", func(t *testing.T) {
				field, found := FindRecursiveField(reflect.TypeOf(value), "Field1")
				require.False(t, found)
				require.Empty(t, field.Name)
			})
			t.Run("Value", func(t *testing.T) {
				value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1")
				require.Error(t, err)
				assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field1"))
				require.Empty(t, value)
			})
		})
		t.Run("bool", func(t *testing.T) {
			value := true
			t.Run("Field", func(t *testing.T) {
				field, found := FindRecursiveField(reflect.TypeOf(value), "Field1")
				require.False(t, found)
				require.Empty(t, field.Name)
			})
			t.Run("Value", func(t *testing.T) {
				value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1")
				require.Error(t, err)
				assert.Contains(t, err.Error(), "Field1")
				require.Empty(t, value)
			})
		})
	})
	t.Run("Will work for struct types", func(t *testing.T) {
		t.Run("TestStruct with all fields", func(t *testing.T) {
			value := TestStruct{
				Substruct: Substruct{
					Subsubstruct: Subsubstruct{
						Subsubfield1: "subsubstruct-subsubfield1-value",
					},
					Field1:    "substruct-field1-value",
					Subfield1: "substruct-subfield1-value",
				},
				Field1: "field1-value",
				Field2: "field2-value",
				Field3: struct {
					Field1 string
					Field2 string
				}{
					Field1: "field3-field1-value",
					Field2: "field3-field2-value",
				},
				Field4: &struct {
					Field1 string
					Field2 string
				}{
					Field1: "field4-field1-value",
					Field2: "field4-field2-value",
				},
			}
			t.Run("Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1")
					require.NoError(t, err)
					require.Equal(t, "field1-value", value.Interface())
				})
			})
			t.Run("Subfield1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Subfield1")
					require.True(t, found)
					require.Equal(t, "Subfield1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Subfield1")
					require.NoError(t, err)
					require.Equal(t, "substruct-subfield1-value", value.Interface())
				})
			})
			t.Run("Subsubfield1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Subsubfield1")
					require.True(t, found)
					require.Equal(t, "Subsubfield1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Subsubfield1")
					require.NoError(t, err)
					require.Equal(t, "subsubstruct-subsubfield1-value", value.Interface())
				})
			})
			t.Run("Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Bogus"))
					require.Empty(t, value)
				})
			})
			t.Run("Field1.Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field1.Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1.Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field1.Bogus"))
					require.Empty(t, value)
				})
			})
			t.Run("Field3.Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field3.Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field3.Field1")
					require.NoError(t, err)
					require.Equal(t, "field3-field1-value", value.Interface())
				})
			})
			t.Run("Field3.Field1.Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field3.Field1.Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field3.Field1.Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field3.Field1.Bogus"))
					require.Empty(t, value)
				})
			})
			t.Run("Field4.Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field4.Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field4.Field1")
					require.NoError(t, err)
					require.Equal(t, "field4-field1-value", value.Interface())
				})
			})
		})
		t.Run("TestStruct with empty and nil fields", func(t *testing.T) {
			value := TestStruct{
				Field1: "field1-value",
				Field2: "field2-value",
			}
			t.Run("Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1")
					require.NoError(t, err)
					require.Equal(t, "field1-value", value.Interface())
				})
			})
			t.Run("Subfield1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Subfield1")
					require.True(t, found)
					require.Equal(t, "Subfield1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Subfield1")
					require.NoError(t, err)
					require.Equal(t, "", value.Interface())
				})
			})
			t.Run("Subsubfield1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Subsubfield1")
					require.True(t, found)
					require.Equal(t, "Subsubfield1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Subsubfield1")
					require.NoError(t, err)
					require.Equal(t, "", value.Interface())
				})
			})
			t.Run("Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Bogus"))
					require.Empty(t, value)
				})
			})
			t.Run("Field1.Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field1.Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1.Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field1.Bogus"))
					require.Empty(t, value)
				})
			})
			t.Run("Field3.Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field3.Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field3.Field1")
					require.NoError(t, err)
					require.Equal(t, "", value.Interface())
				})
			})
			t.Run("Field4.Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field4.Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field4.Field1")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field4")) // We never make it to "Field1", since "Field4" is nil.
					require.Empty(t, value)
				})
			})
			t.Run("Field4.Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field4.Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field4.Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field4.Bogus"))
					require.Empty(t, value)
				})
			})
		})
		t.Run("TestStruct only anonymous struct field", func(t *testing.T) {
			value := TestStruct{
				Substruct: Substruct{
					Subsubstruct: Subsubstruct{
						Subsubfield1: "subsubstruct-subsubfield1-value",
					},
					Field1:    "substruct-field1-value",
					Subfield1: "substruct-subfield1-value",
				},
			}
			t.Run("Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1")
					require.NoError(t, err)
					require.Equal(t, "", value.Interface())
				})
			})
			t.Run("Subfield1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Subfield1")
					require.True(t, found)
					require.Equal(t, "Subfield1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Subfield1")
					require.NoError(t, err)
					require.Equal(t, "substruct-subfield1-value", value.Interface())
				})
			})
			t.Run("Subsubfield1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Subsubfield1")
					require.True(t, found)
					require.Equal(t, "Subsubfield1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Subsubfield1")
					require.NoError(t, err)
					require.Equal(t, "subsubstruct-subsubfield1-value", value.Interface())
				})
			})
			t.Run("Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Bogus"))
					require.Empty(t, value)
				})
			})
			t.Run("Field1.Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field1.Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field1.Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field1.Bogus"))
					require.Empty(t, value)
				})
			})
			t.Run("Field3.Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field3.Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field3.Field1")
					require.NoError(t, err)
					require.Equal(t, "", value.Interface())
				})
			})
			t.Run("Field4.Field1", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field4.Field1")
					require.True(t, found)
					require.Equal(t, "Field1", field.Name)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field4.Field1")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field4")) // We never make it to "Field1", since "Field4" is nil.
					require.Empty(t, value)
				})
			})
			t.Run("Field4.Bogus", func(t *testing.T) {
				t.Run("Field", func(t *testing.T) {
					field, found := FindRecursiveField(reflect.TypeOf(value), "Field4.Bogus")
					require.False(t, found)
					require.Empty(t, field)
				})
				t.Run("Value", func(t *testing.T) {
					value, err := FindRecursiveValue(reflect.ValueOf(value), "Field4.Bogus")
					require.Error(t, err)
					assert.Contains(t, err.Error(), fmt.Sprintf("%q", "Field4.Bogus"))
					require.Empty(t, value)
				})
			})
		})
	})
}
