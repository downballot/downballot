package api

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/apitoken"
	"github.com/downballot/downballot/internal/appconfig"
	"github.com/downballot/downballot/internal/application"
	restfulspec "github.com/emicklei/go-restful-openapi/v2"
	"github.com/emicklei/go-restful/v3"
	"github.com/go-openapi/spec"
	"github.com/sirupsen/logrus"
)

// These are the attributes that are available as part of the request.
const (
	AttributeConfig      = "config"            // (*appconfig.MergedConfig) This is the tenant config.
	AttributeLoggedIn    = "session:logged_in" // (bool) This is true if the session is valid.
	AttributeSystemAdmin = "system:admin"      // (bool) This is true if the user is a system admin.
	AttributeUserAllowed = "user:allowed"      // (bool) This is true if the user is allowed to access the system.
	AttributeUserID      = "user:id"           // (string) This is the user's ID.
	AttributeUserName    = "user:name"         // (string) This is the user's name.
	AttributeUserToken   = "user:token"        // (string) This is the user's API token.
)

// DefaultPageSize is the default page size for paginated things.
const DefaultPageSize = 25

// Instance contains the local data for the API.
type Instance struct {
	App    *application.App // This is the application.
	Config appconfig.Config // This is the full configuration.

	jwtSecret     []byte          // This is the JWT secret, if any.
	jwtPublicKey  *rsa.PublicKey  // This is the JWT public key, if any.
	jwtPrivateKey *rsa.PrivateKey // This is the JWT private key, if any.
}

// New returns a new API instance.
func New() *Instance {
	instance := new(Instance)
	return instance
}

// WriteEntity writes an entity with a 200 status.
func WriteEntity(ctx context.Context, response *restful.Response, value interface{}) {
	WriteHeaderAndEntity(ctx, response, http.StatusOK, value)
}

// WriteHeaderAndEntity writes an enitity with the given status code.
func WriteHeaderAndEntity(ctx context.Context, response *restful.Response, code int, value interface{}) {
	payload, err := json.Marshal(value)
	if err != nil {
		logrus.WithContext(ctx).Warnf("Could not serialize the value to JSON: %v", err)
		return
	}

	envelope := downballotapi.Envelope{
		Success: true,
		Message: "Okay",
		Data:    payload,
	}
	contents, err := json.MarshalIndent(envelope, "", " ")
	if err != nil {
		logrus.WithContext(ctx).Warnf("Could not serialize the envelope to JSON: %v", err)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)
	_, _ = response.Write(contents)
}

// WriteHeaderAndError writes an error with the given status code.
func WriteHeaderAndError(ctx context.Context, response *restful.Response, code int, value error) {
	WriteHeaderAndText(ctx, response, code, value.Error())
}

// WriteHeaderAndText writes text with the given status code.
func WriteHeaderAndText(ctx context.Context, response *restful.Response, code int, value string) {
	envelope := downballotapi.Envelope{
		Success: false,
		Message: value,
	}
	contents, err := json.MarshalIndent(envelope, "", " ")
	if err != nil {
		logrus.WithContext(ctx).Warnf("Could not serialize the envelope to JSON: %v", err)
		return
	}

	response.Header().Set("Content-Type", "application/json")
	response.WriteHeader(code)
	_, _ = response.Write(contents)
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
		ws := new(restful.WebService)
		ws.Path("/api/v1")
		ws.Consumes(restful.MIME_JSON)
		ws.Produces(restful.MIME_JSON)

		// Set up a filter to catch panics and return a 500 error.
		ws.Filter(func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
			defer func() {
				if r := recover(); r != nil {
					logrus.WithContext(req.Request.Context()).Errorf("Recovered from panic: %v", r)
					logrus.WithContext(req.Request.Context()).Errorf("Stack trace:\n%s", debug.Stack())
					WriteHeaderAndText(req.Request.Context(), resp, http.StatusInternalServerError, fmt.Sprintf("%v", r))
				}
			}()

			chain.ProcessFilter(req, resp)
		})

		// Register the various endpoints.
		i.registerAuthenticationEndpoints(ws)

		container.Add(ws)
		documentedWebServices = append(documentedWebServices, ws)
	}

	container.ServiceErrorHandler(func(serviceError restful.ServiceError, request *restful.Request, response *restful.Response) {
		ctx := request.Request.Context()
		logrus.WithContext(ctx).Debugf("Service error: %v", serviceError)
		for header, values := range serviceError.Header {
			for _, value := range values {
				response.Header().Add(header, value)
			}
		}

		WriteHeaderAndText(ctx, response, serviceError.Code, serviceError.Message)
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

// ValidateToken validates an API token (from a string) and returns the token's claims,
// which can be used to learn more about the user.
//
// If the token has expired, then this fails with an error.
func (i *Instance) ValidateToken(tokenString string) (apitoken.TokenClaims, error) {
	var claims apitoken.TokenClaims
	_, err := jwt.ParseWithClaims(tokenString, &claims,
		func(t *jwt.Token) (interface{}, error) {
			if i.jwtSecret != nil {
				return i.jwtSecret, nil
			}
			if i.jwtPublicKey != nil {
				return i.jwtPublicKey, nil
			}
			return jwt.UnsafeAllowNoneSignatureType, nil
		},
	)
	if err != nil {
		return claims, fmt.Errorf("could not parse token: %v", err)
	}

	return claims, nil
}

// doRequireAuthentication can be used as a restful.RouteBuilder `Do` argument to require authentication.
//
// This will set the required headers and return failure modes.
func (i *Instance) doRequireAuthentication(builder *restful.RouteBuilder) {
	builder.Filter(i.requireAuthenticationFilter(true)).
		Param(restful.HeaderParameter("Authorization", "The authorization token.  This should be of the form: \"Bearer ${token}\"")).
		Returns(http.StatusUnauthorized, "Unauthorized", nil)
}

// doAcceptAuthentication can be used as a restful.RouteBuilder `Do` argument to accept (but not require) authentication.
//
// This will set the required headers and return failure modes.
func (i *Instance) doAcceptAuthentication(builder *restful.RouteBuilder) {
	builder.Filter(i.requireAuthenticationFilter(false)).
		Param(restful.HeaderParameter("Authorization", "The authorization token.  This should be of the form: \"Bearer ${token}\"")).
		Returns(http.StatusUnauthorized, "Unauthorized", nil)
}

// doRequireSystemAdministrator can be used as a restful.RouteBuilder `Do` argument to require the user to be a system administrator.
func (i *Instance) doRequireSystemAdministrator(builder *restful.RouteBuilder) {
	fn := func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		ctx := request.Request.Context()

		if request.Attribute(AttributeSystemAdmin) != true {
			WriteHeaderAndText(ctx, response, http.StatusForbidden, "Forbidden")
			return
		}

		// Continue with the rest of the chain.
		chain.ProcessFilter(request, response)
	}
	builder.Filter(fn).
		Returns(http.StatusUnauthorized, "Forbidden", nil)
}

// requireAuthenticationFilter is a `restful` filter that ensures that the user is authenticated.
//
// If `required` is true, then this will fail with a 401 error if no authentication was given.
// If `required` is false, then authentication information will be accepted if provided, but it
// will not fail if not.
//
// The only authentication scheme that we support is "bearer" authentication with a token.
// The token can either be a user token or the "master" token for the service.
func (i *Instance) requireAuthenticationFilter(required bool) func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	return func(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
		ctx := request.Request.Context()

		// Figure out what the token is.
		if headerValue := request.HeaderParameter("Authorization"); headerValue != "" {
			logrus.WithContext(request.Request.Context()).Debugf("[auth] Header Authorization provided.")

			token := ""
			if strings.HasPrefix(headerValue, "Bearer ") {
				token = strings.TrimPrefix(headerValue, "Bearer ")
			} else {
				logrus.WithContext(request.Request.Context()).Infof("[auth] Invalid Authoriation format.")
				WriteHeaderAndText(ctx, response, http.StatusUnauthorized, "Invalid Authorization format")
				return
			}

			request.SetAttribute(AttributeLoggedIn, true)
			request.SetAttribute(AttributeSystemAdmin, false)
			request.SetAttribute(AttributeUserToken, token)

			claims, err := i.ValidateToken(token)
			if err != nil {
				logrus.WithContext(request.Request.Context()).Infof("[auth] Invalid token: %v", err)
				WriteHeaderAndText(ctx, response, http.StatusUnauthorized, "Invalid token")
				return
			}

			request.SetAttribute(AttributeUserID, claims.Subject)
			request.SetAttribute(AttributeUserName, claims.Email)

			// TODO: VALIDATE THE USER

			// If authentication is required, then fail if the user isn't "allowed".
			if required {
				if request.Attribute(AttributeUserAllowed) != true {
					// TODO: Consider a more generic account-disabled error message.
					WriteHeaderAndText(ctx, response, http.StatusForbidden, fmt.Sprintf("You must be logged in."))
					return
				}
			}

			// Continue with the rest of the chain.
			chain.ProcessFilter(request, response)
			return
		}

		// No authorization header was given.
		logrus.WithContext(request.Request.Context()).Debugf("[auth] Missing header: Authorization")
		if !required {
			// Continue with the rest of the chain.
			chain.ProcessFilter(request, response)
		} else {
			WriteHeaderAndText(ctx, response, http.StatusUnauthorized, "Missing header: Authorization")
		}
	}
}
