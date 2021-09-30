package httpextra

import (
	"net/http"
	"strings"
)

// GetRequestHost returns the desired host of the request, after taking into account
// proxies and such.
//
// First priority goes to the `Forwarded` header, if it has `host` set.
// Second priority goes to the `X-Forwarded-Host` header, if there is a value.
// Last priority goes to the Host field in the request.
func GetRequestHost(request *http.Request) string {
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
				if parts[0] == "host" && len(parts[1]) > 0 {
					return parts[1]
				}
			}
		}
	}

	// If the "X-Forwarded-Host" header is set, use that if it gives us anything.
	//
	// Information on the "X-Forwarded-Host" header: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/X-Forwarded-Host
	if headerValue := request.Header.Get("X-Forwarded-Host"); headerValue != "" {
		return headerValue
	}

	// Otherwise, just use whatever the host value is.
	return request.Host
}
