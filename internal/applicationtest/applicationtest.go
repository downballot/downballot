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
	"github.com/tekkamanendless/restapiclient"
	"gorm.io/gorm"
)

// Server is a test instance of the applicaiton.
type Server struct {
	application *application.App
	db          *gorm.DB
	httpServer  *httptest.Server
}

func New(t *testing.T, ctx context.Context) *Server {
	db, err := databasetest.New(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	app := application.New(db)

	myHandler := http.NewServeMux()
	{
		apiInstance := api.New()
		apiInstance.App = app
		apiInstance.Config = api.Config{
			MasterToken: "my-master-token",
		}

		apiContainer := apiInstance.Container(ctx)
		myHandler.Handle("/api/", apiContainer)
	}
	httpServer := httptest.NewServer(myHandler)

	s := &Server{
		application: app,
		db:          db,
		httpServer:  httpServer,
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

// UnauthenticatedClient returns an unauthenticated REST client.
func (s *Server) UnauthenticatedClient() *downballotapi.Client {
	client := downballotapi.New(s.URL())
	return client
}

func (s *Server) AuthenticatedClientMaster() *downballotapi.Client {
	client := downballotapi.New(s.URL(), restapiclient.OptionHeader("Authorization", "Bearer my-master-token"))
	return client
}
