package downballotapi

// ImportPersonResponse is the response from importing persons.
type ImportPersonResponse struct {
	Records uint64 `json:"records"`
}

// ListPersonsResponse is the response from listing the persons.
type ListPersonsResponse struct {
	Persons []*Person `json:"persons"`
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
