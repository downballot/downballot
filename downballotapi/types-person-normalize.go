package downballotapi

// PostPersonNormalizeRequest is the request for normalizing the persons.
type PostPersonNormalizeRequest struct {
	GroupID string `json:"group_id"`
	Filter  string `json:"filter"`
	Pretend bool   `json:"pretend"`
}

// PostPersonNormalizeResponse is the response for normalizing the persons.
type PostPersonNormalizeResponse struct {
	Pretend bool               `json:"pretend"`
	Persons []*NormalizePerson `json:"persons"`
}

// NormalizePerson is a person that was normalized.
type NormalizePerson struct {
	ID        string             `json:"id"`
	VoterID   string             `json:"voter_id"`
	OldFields map[string]*string `json:"old_fields"`
	NewFields map[string]*string `json:"new_fields"`
}
