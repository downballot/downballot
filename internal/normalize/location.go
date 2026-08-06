package normalize

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/cleanup"
	"googlemaps.github.io/maps"
)

type processExpression struct {
	regex       *regexp.Regexp
	replacement string
}

var preprocessExpressions = []processExpression{
	// Fix the Christina Mill apartments.
	// They no longer use apartment numbers.
	{
		regex:       regexp.MustCompile(`^100 christina mill dr apt ([0-9]+), newark, de 19711`),
		replacement: "$1 christina mill dr, newark, de 19711",
	},
	{
		regex:       regexp.MustCompile(`^100 christina mill dr #([0-9]+), newark, de 19711`),
		replacement: "$1 christina mill dr, newark, de 19711",
	},
	// Convert room numbers to apartment numbers.
	{
		regex:       regexp.MustCompile(`^([^,]+) rm ([^ ]+), `),
		replacement: "$1 apt $2, ",
	},
}

var postprocessExpressions = []processExpression{
	// Fix single-letter apartment numbers.
	// Basically, we just want to put a "#" in front of them.
	{
		regex:       regexp.MustCompile(`^([^,]+) ([a-zA-Z]), `),
		replacement: "$1 #$2, ",
	},
	// Fix unconverted apartment numbers.
	// We just want to put a "#" in front of them, instead of "Apt".
	{
		regex:       regexp.MustCompile(`^([^,]+) Apt ([0-9]+[a-zA-Z0-9]*), `),
		replacement: "$1 #$2, ",
	},
}

// PreProcessAddress pre-processes the address.
func PreProcessAddress(address string) string {
	address = strings.ToLower(address)

	for _, expression := range preprocessExpressions {
		if expression.regex.MatchString(address) {
			address = expression.regex.ReplaceAllString(address, expression.replacement)
		}
	}
	return address
}

// PostProcessAddress post-processes the address.
func PostProcessAddress(address string) string {
	for _, expression := range postprocessExpressions {
		if expression.regex.MatchString(address) {
			address = expression.regex.ReplaceAllString(address, expression.replacement)
		}
	}
	return address
}

// Location normalizes the location of a person.
func Location(ctx context.Context, mapsClient *maps.Client, nccClient *NewCastleCountyGIS, normalizePerson *downballotapi.NormalizePerson) error {
	if normalizePerson.OldFields["residential_address"] == nil || *normalizePerson.OldFields["residential_address"] == "" {
		return nil
	}
	address := *normalizePerson.OldFields["residential_address"]

	slog.DebugContext(ctx, fmt.Sprintf("Initial address: %s", address))

	// Pre-process the address.
	address = PreProcessAddress(address)
	slog.DebugContext(ctx, fmt.Sprintf("Pre-processed address: %s", address))

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

					newAddress, err := cleanup.Address(result.FormattedAddress)
					if err != nil {
						slog.WarnContext(ctx, fmt.Sprintf("could not cleanup address: %s", err))
					} else {
						address = newAddress
						normalizePerson.NewFields["residential_address"] = &address
					}
					break
				}
			}
		}
	}

	// Post-process the address.
	address = PostProcessAddress(address)
	slog.DebugContext(ctx, fmt.Sprintf("Post-processed address: %s", address))

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
