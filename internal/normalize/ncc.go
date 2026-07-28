package normalize

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

type NewCastleCountyGIS struct {
	httpClient             http.Client
	parcelIDToCommunityMap map[string]string
}

func (n *NewCastleCountyGIS) LookupCommunity(ctx context.Context, coordinates string) (string, error) {
	if n.parcelIDToCommunityMap == nil {
		n.parcelIDToCommunityMap = map[string]string{}
	}

	var output string

	var xyCoordinates string
	{
		parts := strings.SplitN(coordinates, ",", 2)
		if len(parts) != 2 {
			return output, fmt.Errorf("invalid coordinates: %s", coordinates)
		}
		xyCoordinates = fmt.Sprintf("%s,%s", parts[1], parts[0])
	}

	type GISResponse struct {
		Features []struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"features"`
	}

	var parcelID string
	{
		queryParameters := url.Values{}
		queryParameters.Add("where", "(1=1)")
		queryParameters.Add("outFields", strings.Join([]string{"ADDRESS", "PRCLID"}, ","))
		queryParameters.Add("returnGeometry", "false")
		queryParameters.Add("outSR", "4326")
		queryParameters.Add("f", "json")

		queryParameters.Add("resultRecordCount", "10")

		queryParameters.Add("geometry", xyCoordinates)
		queryParameters.Add("geometryType", "esriGeometryPoint")
		queryParameters.Add("inSR", "4326")
		queryParameters.Add("spatialRel", "esriSpatialRelIntersects")

		targetURL := "https://gis.nccde.org/agsserver/rest/services/BaseMaps/Addresses/MapServer/4/query?" + queryParameters.Encode()
		slog.DebugContext(ctx, fmt.Sprintf("Target URL: %s", targetURL))
		request, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			return output, fmt.Errorf("could not create request: %w", err)
		}
		response, err := n.httpClient.Do(request)
		if err != nil {
			return output, fmt.Errorf("could not do request: %w", err)
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return output, fmt.Errorf("could not read body: %w", err)
		}

		var gisResponse GISResponse
		err = json.Unmarshal(body, &gisResponse)
		if err != nil {
			return output, fmt.Errorf("could not unmarshal body: %w", err)
		}
		slog.DebugContext(ctx, fmt.Sprintf("GIS response: %+v", gisResponse))

		if len(gisResponse.Features) > 0 {
			feature := gisResponse.Features[0]
			subdiv := feature.Attributes["PRCLID"]
			slog.DebugContext(ctx, fmt.Sprintf("PRCLID: %+v", subdiv))
			if v, ok := subdiv.(string); ok {
				parcelID = v
			}
		}
	}

	if parcelID != "" {
		if n.parcelIDToCommunityMap[parcelID] != "" {
			return n.parcelIDToCommunityMap[parcelID], nil
		}

		queryParameters := url.Values{}
		queryParameters.Add("where", fmt.Sprintf("(PARCELID = '%s')", parcelID))
		queryParameters.Add("outFields", strings.Join([]string{"ADDRESS", "COMMUNITY"}, ","))
		queryParameters.Add("returnGeometry", "false")
		queryParameters.Add("outSR", "4326")
		queryParameters.Add("f", "json")

		queryParameters.Add("resultRecordCount", "1")

		targetURL := "https://gis.nccde.org/agsserver/rest/services/BaseMaps/Addresses/MapServer/2/query?" + queryParameters.Encode()
		slog.DebugContext(ctx, fmt.Sprintf("Target URL: %s", targetURL))
		request, err := http.NewRequest("GET", targetURL, nil)
		if err != nil {
			return output, fmt.Errorf("could not create request: %w", err)
		}
		response, err := n.httpClient.Do(request)
		if err != nil {
			return output, fmt.Errorf("could not do request: %w", err)
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return output, fmt.Errorf("could not read body: %w", err)
		}

		var gisResponse GISResponse
		err = json.Unmarshal(body, &gisResponse)
		if err != nil {
			return output, fmt.Errorf("could not unmarshal body: %w", err)
		}
		slog.DebugContext(ctx, fmt.Sprintf("GIS response: %+v", gisResponse))

		if len(gisResponse.Features) > 0 {
			feature := gisResponse.Features[0]
			community := feature.Attributes["COMMUNITY"]
			slog.DebugContext(ctx, fmt.Sprintf("COMMUNITY: %+v", community))
			if v, ok := community.(string); ok {
				n.parcelIDToCommunityMap[parcelID] = v
				output = v
			}
		}
	}

	return output, nil
}
