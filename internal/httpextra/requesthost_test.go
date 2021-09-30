package httpextra

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetRequestHost(t *testing.T) {
	rows := []struct {
		description string
		request     http.Request
		result      string
	}{
		{
			description: "No fancy headers",
			request: http.Request{
				Host: "my-host",
			},
			result: "my-host",
		},
		{
			description: "X-Forwarded-Host set",
			request: http.Request{
				Host: "my-host",
				Header: http.Header{
					"X-Forwarded-Host": []string{"x-host"},
				},
			},
			result: "x-host",
		},
		{
			description: "X-Forwarded-Host empty",
			request: http.Request{
				Host: "my-host",
				Header: http.Header{
					"X-Forwarded-Host": []string{""},
				},
			},
			result: "my-host",
		},
		{
			description: "Forwarded host not set",
			request: http.Request{
				Host: "my-host",
				Header: http.Header{
					"Forwarded": []string{"for=somebody"},
				},
			},
			result: "my-host",
		},
		{
			description: "Forwarded host empty",
			request: http.Request{
				Host: "my-host",
				Header: http.Header{
					"Forwarded": []string{"for=somebody;host="},
				},
			},
			result: "my-host",
		},
		{
			description: "Forwarded",
			request: http.Request{
				Host: "my-host",
				Header: http.Header{
					"Forwarded": []string{"for=somebody;host=f-host"},
				},
			},
			result: "f-host",
		},
		{
			description: "Forwarded has highest priority",
			request: http.Request{
				Host: "my-host",
				Header: http.Header{
					"Forwarded":        []string{"for=somebody;host=f-host"},
					"X-Forwarded-Host": []string{"x-host"},
				},
			},
			result: "f-host",
		},
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			result := GetRequestHost(&row.request)
			assert.Equal(t, row.result, result)
		})
	}
}
