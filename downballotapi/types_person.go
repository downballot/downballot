package downballotapi

// ImportPersonResponse is the response from importing persons.
type ImportPersonResponse struct {
	Records uint64 `json:"records"`
}

// ListPersonsResponse is the response from listing the persons.
type ListPersonsResponse struct {
	Persons []*Person `json:"persons"`
}

// Person is an person.
type Person struct {
	ID      string            `json:"id"`
	VoterID string            `json:"voter_id"`
	Fields  map[string]string `json:"fields"`
}
