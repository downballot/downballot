package downballotapi

// GetGroupPersonCountResponse is the response from getting the group person count.
type GetGroupPersonCountResponse struct {
	Groups []*GroupPersonCount `json:"groups"`
}

// GroupPersonCount is a group person count.
type GroupPersonCount struct {
	ID       string `json:"id"`
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
	Filter   string `json:"filter"`
	Count    int64  `json:"count"`
}
