package httpextra

import (
	"fmt"
	"net"
	"net/http"
	"strings"
)

// GetRequestSource returns the source address of the request, after taking into account
// proxies and such.
func GetRequestSource(request *http.Request) (string, error) {
	// If the "Forwarded" header is set, use that if it gives us anything.
	//
	// Information on the "Forwarded" header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Forwarded
	if headerValue := request.Header.Get("Forwarded"); headerValue != "" {
		headerValue = strings.ToLower(headerValue)
		directives := strings.Split(headerValue, ";")
		for _, directive := range directives {
			directive = strings.TrimSpace(directive)
			multiples := strings.Split(directive, ",")
			for _, multiple := range multiples {
				multiple = strings.TrimSpace(multiple)
				parts := strings.SplitN(multiple, "=", 2)
				if parts[0] == "for" {
					part := strings.Trim(parts[1], "\"")

					host, _, err := net.SplitHostPort(part)
					if err != nil {
						part = strings.TrimPrefix(part, "[")
						part = strings.TrimSuffix(part, "]")
						return part, nil
					}
					return host, nil
				}
			}
		}
	}

	// If the "X-Forwarded-For" header is set, use that if it gives us anything.
	//
	// Information on the "X-Forwarded-For" header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-For
	if headerValue := request.Header.Get("X-Forwarded-For"); headerValue != "" {
		host, _, err := net.SplitHostPort(headerValue)
		if err != nil {
			headerValue = strings.TrimPrefix(headerValue, "[")
			headerValue = strings.TrimSuffix(headerValue, "]")
			return headerValue, nil
		}
		return host, nil
	}

	// Otherwise, just use whatever the remote address is.
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		return "", fmt.Errorf("could not extract the IP portion of %s", request.RemoteAddr)
	}

	return host, nil
}
