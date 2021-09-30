package httpextra

import (
	"context"
	"net/http"
)

// ContextRequestHandler adds the basic request values to the request's context, if available.
func ContextRequestHandler(h http.Handler) http.Handler {
	newHandler := http.NewServeMux()
	newHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		host := GetRequestHost(r)
		if host != "" {
			r = r.WithContext(context.WithValue(r.Context(), ContextKeyHost, host))
		}

		path := r.URL.String()
		if path != "" {
			r = r.WithContext(context.WithValue(r.Context(), ContextKeyPath, path))
		}

		h.ServeHTTP(w, r)
	})
	return newHandler
}
