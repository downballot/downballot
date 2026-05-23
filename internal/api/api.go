package api

import (
	"context"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"runtime/debug"

	"github.com/dgrijalva/jwt-go"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/application"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/tekkamanendless/go-mailer"
	"github.com/tekkamanendless/httperror"
	"github.com/threatmate/restfulwrapper"
)

type API struct {
	jwtSecret     []byte          // This is the JWT secret, if any.
	jwtPublicKey  *rsa.PublicKey  // This is the JWT public key, if any.
	jwtPrivateKey *rsa.PrivateKey // This is the JWT private key, if any.
	mailer        *mailer.Mailer  // This is the mailer.
}

// DefaultPageSize is the default page size for paginated things.
const DefaultPageSize = 25

// Instance contains the local data for the API.
type Instance struct {
	App    *application.App // This is the application.
	Config Config           // This is the full configuration.

	jwtSecret     []byte          // This is the JWT secret, if any.
	jwtPublicKey  *rsa.PublicKey  // This is the JWT public key, if any.
	jwtPrivateKey *rsa.PrivateKey // This is the JWT private key, if any.
}

// New returns a new API instance.
func New() *Instance {
	instance := new(Instance)
	return instance
}

// Container creates a new `restful` container.
//
// If `Debug` is set, then the debug endpoints will be added to it.
func (i *Instance) Container(ctx context.Context) *restful.Container {
	if i.Config.JWTSecret != "" {
		i.jwtSecret = []byte(i.Config.JWTSecret)
	}
	if i.Config.JWTPublicKey != "" {
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(i.Config.JWTPublicKey))
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Could not parse PEM public key: %v", err))
		} else {
			i.jwtPublicKey = publicKey
		}
	}
	if i.Config.JWTPrivateKey != "" {
		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(i.Config.JWTPrivateKey))
		if err != nil {
			slog.WarnContext(ctx, fmt.Sprintf("Could not parse PEM private key: %v", err))
		} else {
			i.jwtPrivateKey = privateKey
		}
	}
	//slog.DebugContext(ctx, fmt.Sprintf("JWT secret: %v", i.jwtSecret))
	//slog.DebugContext(ctx, fmt.Sprintf("JWT public key: %v", i.jwtPublicKey))
	//slog.DebugContext(ctx, fmt.Sprintf("JWT private key: %v", i.jwtPrivateKey))

	container := restful.NewContainer()

	// Set up CORS support.
	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{},
		AllowedHeaders: []string{"Accept", "Authorization", "Content-Type"},
		AllowedDomains: []string{}, // Setting this to the empty array means that everything is allowed.
		CookiesAllowed: false,
		Container:      container,
	}
	container.Filter(cors.Filter)

	// Register the documented endpoints.
	var documentedWebServices []*restful.WebService
	{
		ctx := context.TODO()
		webService := restfulwrapper.WebService("/api/v1").
			Consumes(restful.MIME_JSON).
			Produces(restful.MIME_JSON)

		webService.WebService().Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
			ctx := req.Request.Context()

			defer func() {
				if r := recover(); r != nil {
					slog.ErrorContext(ctx, fmt.Sprintf("Recovered from panic: %v", r))
					slog.ErrorContext(ctx, fmt.Sprintf("Stack trace:\n%s", debug.Stack()))
					wrappedError := downballotwrapper.Error(fmt.Errorf("%v", r))
					writer := wrappedError.(restfulwrapper.ErrorWriter)
					writer.WriteError(resp)
				}
			}()

			chain.ProcessFilter(req, resp)
		})

		webService.ErrorHandler(func(err error) error {
			// Wrap the error so that it uses our envelope.
			return downballotwrapper.Error(err)
		})

		{
			middlewareConfig := downballotwrapper.Config{
				DB:          i.App.DB(),
				SystemToken: i.Config.MasterToken,
			}
			if i.Config.JWTSecret != "" {
				middlewareConfig.JWTSecret = []byte(i.Config.JWTSecret)
			}

			var mailerInstance *mailer.Mailer
			if i.Config.SendGridAPIKey != "" {
				mailerInstance = mailer.New(mailer.TypeSendgrid, mailer.WithAPIKey(i.Config.SendGridAPIKey))
			}

			session := webService.Session().
				Attributes(middlewareConfig.Attributes()).
				Do(middlewareConfig.Do())
			session.Register(ctx, "/", &API{
				jwtSecret:     i.jwtSecret,
				jwtPublicKey:  i.jwtPublicKey,
				jwtPrivateKey: i.jwtPrivateKey,
				mailer:        mailerInstance,
			})
		}

		documentedWebServices = append(documentedWebServices, webService.WebService())

		container.Add(webService.WebService())
	}

	container.ServiceErrorHandler(func(serviceError restful.ServiceError, req *restful.Request, resp *restful.Response) {
		ctx := req.Request.Context()

		slog.DebugContext(ctx, fmt.Sprintf("Service error: %v", serviceError))
		for header, values := range serviceError.Header {
			for _, value := range values {
				resp.Header().Add(header, value)
			}
		}

		err := fmt.Errorf("%w: %s", httperror.ErrorFromStatus(serviceError.Code), serviceError.Message)
		wrappedError := downballotwrapper.Error(err)
		writer := wrappedError.(restfulwrapper.ErrorWriter)
		writer.WriteError(resp)
	})

	config := restfulspec.Config{
		// This is what will be used to build the service list.
		WebServices: documentedWebServices,
		// This is where the docs will live.
		APIPath: "/api/swagger.json",
		// This is a post-processing hook.
		PostBuildSwaggerObjectHandler: func(swo *spec.Swagger) {
			swo.Info = &spec.Info{
				InfoProps: spec.InfoProps{
					Title:       "Downballot",
					Description: "Downballot API",
					Contact: &spec.ContactInfo{
						ContactInfoProps: spec.ContactInfoProps{
							Name:  "Downballot Support",
							Email: "support@downballot.io",
							URL:   "https://www.downballot.io",
						},
					},
					/*
						License: &spec.License{
							Name: "MIT",
							URL:  "http://mit.org",
						},
					*/
					//Version: build.Version,
				},
			}
		},
	}
	container.Add(restfulspec.NewOpenAPIService(config))

	return container
}
