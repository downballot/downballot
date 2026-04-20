package restcsv

import (
	"encoding/csv"
	"fmt"

	"github.com/emicklei/go-restful/v3"
)

type Table struct {
	Header []string
	Rows   [][]string
}

// String returns a simple summary string to prevent giant log messages from being generated.
func (t Table) String() string {
	return fmt.Sprintf("%T(%d columns and %d rows)", t, len(t.Header), len(t.Rows))
}

type EntityReaderWriter struct{}

var _ restful.EntityReaderWriter = (*EntityReaderWriter)(nil)

func init() {
	restful.RegisterEntityAccessor("text/csv", &EntityReaderWriter{})
}

// Read a serialized version of the value from the request.
// The Request may have a decompressing reader. Depends on Content-Encoding.
func (erw *EntityReaderWriter) Read(req *restful.Request, v any) error {
	csvReader := csv.NewReader(req.Request.Body)
	records, err := csvReader.ReadAll()
	if err != nil {
		return fmt.Errorf("could not read CSV: %w", err)
	}

	var table Table
	if len(records) > 0 {
		table.Header = records[0]
		records = records[1:]
	}
	table.Rows = records

	switch typedValue := v.(type) {
	case *Table:
		*typedValue = table
	default:
		return fmt.Errorf("invalid type:%T", typedValue)
	}
	return nil
}

// Write a serialized version of the value on the response.
// The Response may have a compressing writer. Depends on Accept-Encoding.
// status should be a valid Http Status code
func (erw *EntityReaderWriter) Write(resp *restful.Response, status int, v any) error {
	// TODO:
	return nil
}
