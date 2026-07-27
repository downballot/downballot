package downballotapi

// PostMapGeocodeRequest is the request body for the POST /map/geocode endpoint.
type PostMapGeocodeRequest struct {
	Address string `json:"address"`
}

// PostMapGeocodeResponse is the response body for the POST /map/geocode endpoint.
type PostMapGeocodeResponse struct {
	Results []MapGeocodeResult `json:"results"`
}

// MapGeocodeResult is a result from the map/geocode endpoint.
type MapGeocodeResult struct {
	FormattedAddress string      `json:"formatted_address"`
	Coordinates      Coordinates `json:"coordinates"`
}

// Coordinates is a coordinate on Earth.
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
