package downballotapi

// HealthCheckResponse is the response from creating a group
type HealthCheckResponse struct {
	Healthy bool `json:"healthy"`
}
