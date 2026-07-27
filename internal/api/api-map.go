package api

import (
	"context"
	"fmt"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/tekkamanendless/restfulwrapper"
	"googlemaps.github.io/maps"
)

type PostMapGeocodeMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.UseDatabase
	downballotwrapper.RequireAuthenticatedUser
	_    string                              `api:"httppath:/map/geocode"`
	_    string                              `api:"doc" description:"Search for a location."`
	_    string                              `api:"notes" description:"This searches for a location"`
	Body downballotapi.PostMapGeocodeRequest `api:"body"`
}

func (a *API) PostMapGeocode(ctx context.Context, meta PostMapGeocodeMetadata) (output downballotapi.Envelope[downballotapi.PostMapGeocodeResponse], err error) {
	client, err := maps.NewClient(maps.WithAPIKey(a.googleMapsServerAPIKey))
	if err != nil {
		return output, fmt.Errorf("could not create Google Maps client: %w", err)
	}

	geocodeRequest := &maps.GeocodingRequest{
		Address: meta.Body.Address,
	}
	results, err := client.Geocode(ctx, geocodeRequest)
	if err != nil {
		return output, fmt.Errorf("could not geocode address: %w", err)
	}

	output.Data.Results = make([]downballotapi.MapGeocodeResult, len(results))
	for i, result := range results {
		output.Data.Results[i] = downballotapi.MapGeocodeResult{
			FormattedAddress: result.FormattedAddress,
			Coordinates: downballotapi.Coordinates{
				Latitude:  result.Geometry.Location.Lat,
				Longitude: result.Geometry.Location.Lng,
			},
		}
	}

	return output, nil
}
