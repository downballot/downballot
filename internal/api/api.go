package api

import (
	"context"
	"crypto/rsa"
	"fmt"
	"runtime/debug"

	"github.com/dgrijalva/jwt-go"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/application"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/sirupsen/logrus"
	"github.com/tekkamanendless/httperror"
	"github.com/threatmate/restfulwrapper"
)

type API struct {
	App *application.App

	jwtSecret     []byte          // This is the JWT secret, if any.
	jwtPublicKey  *rsa.PublicKey  // This is the JWT public key, if any.
	jwtPrivateKey *rsa.PrivateKey // This is the JWT private key, if any.
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
func (i *Instance) Container() *restful.Container {
	if i.Config.JWTSecret != "" {
		i.jwtSecret = []byte(i.Config.JWTSecret)
	}
	if i.Config.JWTPublicKey != "" {
		publicKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(i.Config.JWTPublicKey))
		if err != nil {
			logrus.Warnf("Could not parse PEM public key: %v", err) // We don't have a context here.  This happens at initialization.
		} else {
			i.jwtPublicKey = publicKey
		}
	}
	if i.Config.JWTPrivateKey != "" {
		privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(i.Config.JWTPrivateKey))
		if err != nil {
			logrus.Warnf("Could not parse PEM private key: %v", err) // We don't have a context here.  This happens at initialization.
		} else {
			i.jwtPrivateKey = privateKey
		}
	}
	//logrus.Debugf("JWT secret: %v", i.jwtSecret) // We don't have a context here.  This happens at initialization.
	//logrus.Debugf("JWT public key: %v", i.jwtPublicKey) // We don't have a context here.  This happens at initialization.
	//logrus.Debugf("JWT private key: %v", i.jwtPrivateKey) // We don't have a context here.  This happens at initialization.

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
					logrus.WithContext(ctx).Errorf("Recovered from panic: %v", r)
					logrus.WithContext(ctx).Errorf("Stack trace:\n%s", debug.Stack())
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

			session := webService.Session().
				Attributes(middlewareConfig.Attributes()).
				Do(middlewareConfig.Do())
			session.Register(ctx, "/", &API{
				App:           i.App,
				jwtSecret:     i.jwtSecret,
				jwtPublicKey:  i.jwtPublicKey,
				jwtPrivateKey: i.jwtPrivateKey,
			})
		}

		documentedWebServices = append(documentedWebServices, webService.WebService())

		container.Add(webService.WebService())
	}

	container.ServiceErrorHandler(func(serviceError restful.ServiceError, req *restful.Request, resp *restful.Response) {
		ctx := req.Request.Context()

		logrus.WithContext(ctx).Debugf("Service error: %v", serviceError)
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
