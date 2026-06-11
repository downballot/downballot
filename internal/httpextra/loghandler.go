package httpextra

import (
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"
)

// DebugRequest can be used to log additional information about each request.
var DebugRequest bool = false

// LogHandler logs the request before and after it is handled.
func LogHandler(name string, h http.Handler) http.Handler {
	newHandler := http.NewServeMux()
	newHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		requestSource, err := GetRequestSource(r)
		if err != nil {
			requestSource = "unknown"
		}

		host := GetRequestHost(r)

		slog.InfoContext(ctx, fmt.Sprintf("[%s] request-in: %s %s %s %s", name, requestSource, r.Method, host, r.URL.Path))
		betterResponseWriter := MakeBetterResponseWriter(w)

		if DebugRequest {
			slog.InfoContext(ctx, fmt.Sprintf("[%s] URL: %v", name, r.URL))
			slog.InfoContext(ctx, fmt.Sprintf("[%s] Host: %s", name, host))
			slog.InfoContext(ctx, fmt.Sprintf("[%s] Request source: %s", name, requestSource))
			headers := []string{}
			for key := range r.Header {
				headers = append(headers, key)
			}
			slices.Sort(headers)
			slog.InfoContext(ctx, fmt.Sprintf("[%s] Headers: (%d)", name, len(headers)))
			for _, key := range headers {
				for _, value := range r.Header.Values(key) {
					slog.InfoContext(ctx, fmt.Sprintf("[%s] * %s: %v", name, key, value))
				}
			}
		}

		startTime := time.Now()
		h.ServeHTTP(betterResponseWriter, r)
		duration := time.Since(startTime)

		slog.InfoContext(ctx, fmt.Sprintf("[%s] request-out: %s %s %s %s %d %d %v", name, requestSource, r.Method, host, r.URL.Path, betterResponseWriter.StatusCode, betterResponseWriter.BytesWritten, duration))
	})
	return newHandler
}
