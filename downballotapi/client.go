package downballotapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
)

// Client is the client.
type Client struct {
	Context context.Context `json:"-"`       // The context.
	Address string          `json:"address"` // The base address.
	Token   string          `json:"token"`   // The token.
}

// New returns a new Client.
func New(ctx context.Context) *Client {
	return &Client{
		Context: ctx,
	}
}

// makeRequest makes a request to the server.
//
// If `requestPayload` is not nil, then it will be encoded as JSON.
// If `responsePayload` is not nil, then the result will be decoded as JSON into this structure.
func (c *Client) makeRequest(method string, url string, headers map[string]string, token string, requestPayload interface{}, responsePayload interface{}) (int, error) {
	httpClient := http.Client{}

	var requestPayloadReader io.Reader
	if requestPayload != nil {
		contents, err := json.Marshal(requestPayload)
		if err != nil {
			return 0, err
		}
		requestPayloadReader = bytes.NewReader(contents)
	}

	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		basePath := c.Address
		if !strings.HasPrefix(basePath, "https://") && !strings.HasPrefix(basePath, "http://") {
			basePath = "https://" + basePath
		}
		url = strings.TrimRight(basePath, "/") + "/" + strings.TrimLeft(url, "/")
	}

	logrus.WithContext(c.Context).Infof("%s %s", method, url)
	request, err := http.NewRequest(method, url, requestPayloadReader)
	if err != nil {
		return 0, err
	}

	if len(token) > 0 {
		request.Header.Set("Authorization", "Bearer "+token)
	}

	if requestPayload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	for name, value := range headers {
		request.Header.Set(name, value)
	}

	response, err := httpClient.Do(request)
	if err != nil {
		return 0, err
	}

	if response.StatusCode < 200 || response.StatusCode > 299 {
		return response.StatusCode, fmt.Errorf("status code %d", response.StatusCode)
	}
	if responsePayload != nil {
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return response.StatusCode, err
		}
		if v, okay := responsePayload.(*[]byte); okay {
			*v = contents
			return response.StatusCode, nil
		}

		var envelope Envelope
		err = json.Unmarshal(contents, &envelope)
		if err != nil {
			return response.StatusCode, err
		}

		err = json.Unmarshal(envelope.Data, &responsePayload)
		if err != nil {
			return response.StatusCode, err
		}
	}

	return response.StatusCode, nil
}

// Login to the system.
func (c *Client) Login(input *LoginRequest) (*LoginResponse, error) {
	path := "/api/v1/authentication/login"

	var output LoginResponse
	_, err := c.makeRequest(http.MethodPost, path, nil, "", input, &output)
	if err != nil {
		return nil, err
	}
	return &output, nil
}
