package normalize

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/cleanup"
	"googlemaps.github.io/maps"
)

// Location normalizes the location of a person.
func Location(ctx context.Context, mapsClient *maps.Client, nccClient *NewCastleCountyGIS, normalizePerson *downballotapi.NormalizePerson) error {
	if normalizePerson.OldFields["residential_address"] == nil || *normalizePerson.OldFields["residential_address"] == "" {
		return nil
	}
	address := *normalizePerson.OldFields["residential_address"]

	// Google Maps.
	{
		geocodeRequest := &maps.GeocodingRequest{
			Address: address,
		}
		results, err := mapsClient.Geocode(ctx, geocodeRequest)
		if err != nil {
			return fmt.Errorf("could not geocode address: %w", err)
		}
		if len(results) > 0 {
			if len(results) > 1 {
				slog.WarnContext(ctx, fmt.Sprintf("multiple results for address: %s", address))
			} else {
				for _, result := range results {
					if result.PartialMatch {
						continue
					}

					coordinates := fmt.Sprintf("%f,%f", result.Geometry.Location.Lat, result.Geometry.Location.Lng)
					normalizePerson.NewFields["coordinates"] = &coordinates

					address, err := cleanup.Address(result.FormattedAddress)
					if err != nil {
						slog.WarnContext(ctx, fmt.Sprintf("could not cleanup address: %s", err))
					} else {
						normalizePerson.NewFields["residential_address"] = &address
					}
					break
				}
			}
		}
	}

	// New Castle County.
	if normalizePerson.NewFields["coordinates"] != nil && *normalizePerson.NewFields["coordinates"] != "" {
		community, err := nccClient.LookupCommunity(ctx, *normalizePerson.NewFields["coordinates"])
		if err != nil {
			return fmt.Errorf("could not lookup community: %w", err)
		}
		slog.DebugContext(ctx, fmt.Sprintf("Community: %s", community))
		normalizePerson.NewFields["residential_address_development"] = &community
	}

	return nil
}
