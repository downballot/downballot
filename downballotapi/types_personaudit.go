package downballotapi

import (
	"time"

	"github.com/downballot/downballot/internal/api/restcsv"
	"github.com/downballot/downballot/internal/api/resttype"
)

// ListPersonAuditsResponse is the response from listing the persons.
type ListPersonAuditsResponse struct {
	Audits []*PersonAudit `json:"audits"`
}

var _ CSVMarshaler = (*ListPersonAuditsResponse)(nil)

func (r ListPersonAuditsResponse) MarshallCSV() (restcsv.Table, error) {
	table := restcsv.Table{
		Header: []string{"timestamp", "old_value", "new_value"},
		Rows:   make([][]string, 0, len(r.Audits)),
	}

	for _, audit := range r.Audits {
		row := make([]string, len(table.Header))
		row[0] = time.Time(audit.Timestamp).Format(time.RFC3339)
		if audit.OldValue != nil {
			row[1] = *audit.OldValue
		}
		if audit.NewValue != nil {
			row[2] = *audit.NewValue
		}
		table.Rows = append(table.Rows, row)
	}
	return table, nil
}

// PersonAudit is an person audit.
type PersonAudit struct {
	ID        string            `json:"id"`
	VoterID   string            `json:"voter_id"`
	Timestamp resttype.DateTime `json:"timestamp"`
	OldValue  *string           `json:"old_value"`
	NewValue  *string           `json:"new_value"`
}
