package httpextra

// ContextKey is the type for the context key.
// The Go docs recommend not using any built-in type for context keys in order
// to ensure that there are no collisions:
//    https://golang.org/pkg/context/#WithValue
type ContextKey string

// ContextKey constants.
const (
	ContextKeyHost ContextKey = "host" // This is the key for the host.
	ContextKeyPath ContextKey = "path" // This is the key for the path.
)
