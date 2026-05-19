package downballotapi

import "github.com/downballot/downballot/internal/schema"

type PersonFieldDefinitionType schema.PersonFieldDefinitionType

// CreatePersonFieldRequest is the request to create a person field.
type CreatePersonFieldRequest struct {
	Name          string                    `json:"name"`
	Type          PersonFieldDefinitionType `json:"type"`
	AllowEmpty    bool                      `json:"allow_empty"`
	AllowedValues []string                  `json:"allowed_values"`
	AllowedRegex  string                    `json:"allowed_regex"`
}

// CreatePersonFieldResponse is the response from creating a person field.
type CreatePersonFieldResponse struct {
	PersonField PersonField `json:"person_field"`
}

// ListPersonFieldsResponse is the response from listing the person fields.
type ListPersonFieldsResponse struct {
	PersonFields []*PersonField `json:"person_fields"`
}

// GetPersonFieldResponse is the response from getting the person field.
type GetPersonFieldResponse struct {
	PersonField *PersonField `json:"person_field"`
}

// PatchPersonFieldRequest is the request for patching the person field.
type PatchPersonFieldRequest struct {
	Name          *string                    `json:"name"`
	Type          *PersonFieldDefinitionType `json:"type"`
	AllowEmpty    *bool                      `json:"allow_empty"`
	AllowedValues []string                   `json:"allowed_values"`
	AllowedRegex  *string                    `json:"allowed_regex"`
}

// PatchPersonFieldResponse is the response from patching the person field.
type PatchPersonFieldResponse struct {
	PersonField PersonField `json:"person_field"`
}

// PersonField is a person field.
type PersonField struct {
	ID            string                    `json:"id"`
	Name          string                    `json:"name"`
	Type          PersonFieldDefinitionType `json:"type"`
	AllowEmpty    bool                      `json:"allow_empty"`
	AllowedValues []string                  `json:"allowed_values"`
	AllowedRegex  string                    `json:"allowed_regex"`
}
