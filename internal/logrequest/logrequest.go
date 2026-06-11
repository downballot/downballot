package logrequest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
)

// Log the HTTP request details.
//
// This is helpful for debugging.
func Log(ctx context.Context, r *http.Request, logBody bool) {
	slog.InfoContext(ctx, fmt.Sprintf("Protocol: %s", r.Proto))
	slog.InfoContext(ctx, fmt.Sprintf("Host: %s", r.Host))
	slog.InfoContext(ctx, fmt.Sprintf("Method: %s", r.Method))
	slog.InfoContext(ctx, fmt.Sprintf("Request: %s", r.RequestURI))

	headerNames := make([]string, 0, len(r.Header))
	for headerName := range r.Header {
		headerNames = append(headerNames, headerName)
	}
	slices.Sort(headerNames)
	slog.InfoContext(ctx, fmt.Sprintf("Headers: (%d)", len(r.Header)))
	for _, headerName := range headerNames {
		for _, value := range r.Header[headerName] {
			slog.InfoContext(ctx, fmt.Sprintf("* %s: %s", headerName, value))
		}
	}

	if logBody {
		// Read the body.
		contents, _ := io.ReadAll(r.Body)
		// Put the body back the way that we found it.
		r.Body = io.NopCloser(bytes.NewReader(contents))

		slog.InfoContext(ctx, fmt.Sprintf("Body: (%d)", len(contents)))
		if len(contents) > 500 {
			slog.InfoContext(ctx, fmt.Sprintf("%s[truncated]", string(contents[:500])))
		} else {
			slog.InfoContext(ctx, fmt.Sprintf("%s", string(contents)))
		}
	}
}
