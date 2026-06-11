package downballotapi

import (
	"encoding/json"
	"fmt"

	"github.com/downballot/downballot/internal/api/restcsv"
)

// Envelope wraps all responses.
type Envelope[T any] struct {
	Message string `json:"message" description:"This is the error message."`
	Success bool   `json:"success" description:"This will always be false to indicate an error."`
	Data    T      `json:"data,omitempty" description:"This is the contents of the response."`
}

// CSVMarshaler should be implemented by the `Data` content *within* the envelope in order to be
// rendered as CSV.
type CSVMarshaler interface {
	MarshallCSV() (restcsv.Table, error)
}

// WriteCSV implements the restcsv.CSVWriter interface to allow the main envelope response type to
// render to CSV.
func (e Envelope[T]) WriteCSV() ([]byte, error) {
	if marshaler, ok := any(e.Data).(CSVMarshaler); ok {
		table, err := marshaler.MarshallCSV()
		if err != nil {
			return nil, err
		}
		return table.ToCSV()
	}
	return nil, fmt.Errorf("could not marshal the CSV data")
}

// RawEnvelope represents an envelope with a raw JSON payload.
type RawEnvelope Envelope[json.RawMessage]
