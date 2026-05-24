package downballotapi

// CreateFilterRequest is the request to create a group.
type CreateFilterRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Filter      string  `json:"filter"`
	UserID      *string `json:"user_id"`
}

// CreateFilterResponse is the response from creating a group
type CreateFilterResponse Filter

// ListFiltersResponse is the response from listing the groups.
type ListFiltersResponse struct {
	Filters []*Filter `json:"filters"`
}

// GetFilterResponse is the response from getting the group.
type GetFilterResponse struct {
	Filter *Filter `json:"filter"`
}

// Filter is a group.
type Filter struct {
	ID          string  `json:"id"`
	UserID      *string `json:"user_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Filter      string  `json:"filter"`
}

// PatchFilterRequest is the request for patching the group.
type PatchFilterRequest struct {
	UserID      *string `json:"user_id"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Filter      *string `json:"filter"`
}

// PatchFilterResponse is the response from patching the group.
type PatchFilterResponse Filter
