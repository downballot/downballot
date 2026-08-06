package api

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/normalize"
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

	slog.DebugContext(ctx, fmt.Sprintf("Initial address: %s", meta.Body.Address))

	// Pre-process the address.
	address := normalize.PreProcessAddress(meta.Body.Address)
	slog.DebugContext(ctx, fmt.Sprintf("Pre-processed address: %s", address))

	geocodeRequest := &maps.GeocodingRequest{
		Address: address,
	}
	results, err := client.Geocode(ctx, geocodeRequest)
	if err != nil {
		return output, fmt.Errorf("could not geocode address: %w", err)
	}

	output.Data.Results = make([]downballotapi.MapGeocodeResult, 0, len(results))
	for _, result := range results {
		for _, addressComponent := range result.AddressComponents {
			slog.DebugContext(ctx, fmt.Sprintf("Address component: %+v", addressComponent))
		}

		postProcessedAddress := normalize.PostProcessAddress(result.FormattedAddress)
		slog.DebugContext(ctx, fmt.Sprintf("Post-processed address: %s", postProcessedAddress))

		newRecord := downballotapi.MapGeocodeResult{
			InitialAddress:       meta.Body.Address,
			PreProcessedAddress:  address,
			FormattedAddress:     result.FormattedAddress,
			PostProcessedAddress: postProcessedAddress,
			Coordinates: downballotapi.Coordinates{
				Latitude:  result.Geometry.Location.Lat,
				Longitude: result.Geometry.Location.Lng,
			},
		}
		output.Data.Results = append(output.Data.Results, newRecord)
	}

	return output, nil
}
