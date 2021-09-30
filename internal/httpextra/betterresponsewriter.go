package httpextra

import "net/http"

// BetterResponseWriter is a wrapper around `http.ResponseWriter` that keeps track
// of the bytes written and status code.
type BetterResponseWriter struct {
	BytesWritten int
	StatusCode   int
	Writer       http.ResponseWriter
}

// Header is a pass-through to `http.ResponseWriter`'s `Header` function.
func (w *BetterResponseWriter) Header() http.Header {
	return w.Writer.Header()
}

// Write is a pass-through to `http.ResponseWriter`'s `Write` function.
func (w *BetterResponseWriter) Write(b []byte) (int, error) {
	w.BytesWritten += len(b)
	return w.Writer.Write(b)
}

// WriteHeader is a pass-through to `http.ResponseWriter`'s `WriteHeader` function.
func (w *BetterResponseWriter) WriteHeader(statusCode int) {
	w.StatusCode = statusCode
	w.Writer.WriteHeader(statusCode)
}

// MakeBetterResponseWriter creates a new BetterResponseWriter.
func MakeBetterResponseWriter(w http.ResponseWriter) *BetterResponseWriter {
	return &BetterResponseWriter{
		Writer: w,
	}
}
