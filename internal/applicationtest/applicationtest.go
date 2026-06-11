package applicationtest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api"
	"github.com/downballot/downballot/internal/application"
	"github.com/downballot/downballot/internal/databasetest"
	"github.com/stretchr/testify/require"
	"github.com/tekkamanendless/go-mailer"
	"github.com/tekkamanendless/restapiclient"
	"gorm.io/gorm"
)

// Server is a test instance of the applicaiton.
type Server struct {
	application *application.App
	db          *gorm.DB
	httpServer  *httptest.Server
	config      api.Config // This is the configuration for the API.
}

// ContextHandler is a handler that adds modifies the context.Context of the request.
type ContextHandler struct {
	contextFunction func(context.Context) context.Context
	handler         http.Handler
}

// ServeHTTP serves the request.
func (h *ContextHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.contextFunction(r.Context())
	h.handler.ServeHTTP(w, r.WithContext(ctx))
}

// NewContextHandler creates a new ContextHandler.
func NewContextHandler(handler http.Handler, contextFunction func(context.Context) context.Context) *ContextHandler {
	return &ContextHandler{
		contextFunction: contextFunction,
		handler:         handler,
	}
}

func New(t *testing.T, ctx context.Context) *Server {
	db, err := databasetest.New(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	app := application.New(ctx, db)

	apiConfig := api.Config{
		MasterToken:    "my-master-token",
		SendGridAPIKey: "my-sendgrid-api-key",
	}

	myHandler := http.NewServeMux()
	{
		apiInstance := api.New()
		apiInstance.App = app
		apiInstance.Config = apiConfig

		apiContainer := apiInstance.Container(ctx)
		myHandler.Handle("/api/", apiContainer)
	}
	contextHandler := NewContextHandler(myHandler, func(ctx context.Context) context.Context {
		return mailer.WithDummyMode(ctx)
	})
	httpServer := httptest.NewServer(contextHandler)

	s := &Server{
		application: app,
		db:          db,
		httpServer:  httpServer,
		config:      apiConfig,
	}
	return s
}

// Close down the server.
func (s *Server) Close() {
	s.httpServer.Close()
}

// DB returns a database handle.
func (s *Server) DB() *gorm.DB {
	return s.db.Session(&gorm.Session{})
}

// URL returns the URL of the web server.
func (s *Server) URL() string {
	return s.httpServer.URL
}

// Config returns the configuration for the API.
func (s *Server) Config() api.Config {
	return s.config
}

// UnauthenticatedClient returns an unauthenticated REST client.
func (s *Server) UnauthenticatedClient() *downballotapi.Client {
	client := downballotapi.New(s.URL())
	return client
}

func (s *Server) AuthenticatedClientMaster() *downballotapi.Client {
	client := downballotapi.New(s.URL(), restapiclient.OptionHeader("Authorization", "Bearer my-master-token"))
	return client
}
