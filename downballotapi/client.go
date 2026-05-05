package downballotapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/tekkamanendless/restapiclient"
)

// Client is the client.
type Client struct {
	client *restapiclient.Client
}

// New returns a new Client.
func New(path string, options ...restapiclient.Option) *Client {
	return &Client{
		client: restapiclient.New(path, options...),
	}
}

// makeRequest makes a request to the server.
//
// If `requestPayload` is not nil, then it will be encoded as JSON.
// If `responsePayload` is not nil, then the result will be decoded as JSON into this structure.
func (c *Client) Do(ctx context.Context, method string, path string, requestPayload any, responsePayload any, options ...restapiclient.Option) error {
	switch responsePayload.(type) {
	case *restapiclient.RawBytes:
		err := c.client.Do(ctx, method, path, requestPayload, responsePayload, options...)
		if err != nil {
			return fmt.Errorf("could not perform request: %w", err)
		}
	default:
		var envelope RawEnvelope
		err := c.client.Do(ctx, method, path, requestPayload, &envelope, options...)
		if err != nil {
			return fmt.Errorf("could not perform request: %w", err)
		}

		if responsePayload != nil {
			err = json.Unmarshal(envelope.Data, &responsePayload)
			if err != nil {
				return fmt.Errorf("could not unmarshal payload: %w", err)
			}
		}
	}
	return nil
}

// Login to the system.
func (c *Client) Login(ctx context.Context, input *LoginRequest) error {
	var output LoginResponse
	err := c.Do(ctx, http.MethodPost, "/api/v1/authentication/login", input, &output)
	if err != nil {
		return fmt.Errorf("could not log in: %w", err)
	}

	c.client = c.client.WithOptions(restapiclient.OptionHeader("Authorization", "Bearer "+output.Token))
	return nil
}
