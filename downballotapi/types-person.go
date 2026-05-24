package downballotapi

import (
	"maps"
	"slices"

	"github.com/downballot/downballot/internal/api/restcsv"
)

// ImportPersonResponse is the response from importing persons.
type ImportPersonResponse struct {
	Records uint64 `json:"records"`
}

// ListPersonsResponse is the response from listing the persons.
type ListPersonsResponse struct {
	Persons []*Person `json:"persons"`
}

var _ CSVMarshaler = (*ListPersonsResponse)(nil)

func (r ListPersonsResponse) MarshallCSV() (restcsv.Table, error) {
	table := restcsv.Table{
		Header: []string{},
		Rows:   make([][]string, 0, len(r.Persons)),
	}
	{
		headerSet := map[string]bool{}
		for _, person := range r.Persons {
			for name := range person.Fields {
				headerSet[name] = true
			}
		}
		table.Header = slices.Collect(maps.Keys(headerSet))
		slices.Sort(table.Header)
	}

	for _, person := range r.Persons {
		row := make([]string, len(table.Header))
		if person.Fields != nil {
			for i, name := range table.Header {
				row[i] = person.Fields[name]
			}
		}
		table.Rows = append(table.Rows, row)
	}
	return table, nil
}

// GetPersonResponse is the response from getting the person.
type GetPersonResponse struct {
	Person *Person `json:"person"`
}

// PatchPersonRequest is the request for patching the person.
type PatchPersonRequest struct {
	Fields map[string]*string `json:"fields"` // If a field is nil, then it should be removed.
}

// Person is an person.
type Person struct {
	ID      string            `json:"id"`
	VoterID string            `json:"voter_id"`
	Fields  map[string]string `json:"fields"`
}
