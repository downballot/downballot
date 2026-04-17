package downballotwrapper

import (
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"strings"

	"github.com/downballot/downballot/internal/schema"
	"github.com/emicklei/go-restful/v3"
	"github.com/tekkamanendless/httperror"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

type RequireAuthenticatedUser struct {
	CurrentUserID string `api:"downballot.currentUserID"`
}

const LocalMapAuthentication = "downballot.authentication"

// init registers the custom ThreatMate user-related tags for `restfulwrapper`.
func init() {
	restfulwrapper.Register("downballot.currentUserID", func(apiTagValue string, field reflect.StructField, info *restfulwrapper.RestfulFunctionInfo) (restfulwrapper.InputFieldFunction, error) {
		requireAuthentication := false
		switch field.Type.String() {
		case "string":
			requireAuthentication = true
		case "*string":
		default:
			return nil, fmt.Errorf("bad type for field %s", field.Name)
		}

		if info.LocalMap[LocalMapAuthentication] == "" {
			info.Do = append(info.Do, doRequireAuthentication(requireAuthentication))
			info.LocalMap[LocalMapAuthentication] = "true"
		}

		return func(v reflect.Value, req *restful.Request, metadataValue reflect.Value) error {
			ctx := req.Request.Context()

			userID, err := getUserIDFromRequest(req)
			if err != nil {
				slog.DebugContext(ctx, fmt.Sprintf("Could not get current user: %v", err))
				switch v.Interface().(type) {
				case string:
					return restfulwrapper.NewAPIResponseError(http.StatusForbidden, "Forbidden")
				case *string:
					v.Set(reflect.ValueOf((*string)(nil)))
				default:
					return restfulwrapper.NewAPIResponseError(http.StatusInternalServerError, fmt.Sprintf("Bad type for field %s", field.Name))
				}
			} else {
				switch v.Interface().(type) {
				case string:
					v.Set(reflect.ValueOf(userID))
				case *string:
					v.Set(reflect.ValueOf(&userID))
				default:
					return restfulwrapper.NewAPIResponseError(http.StatusInternalServerError, fmt.Sprintf("Bad type for field %s", field.Name))
				}
			}
			return nil
		}, nil
	})
}

// getUserIDFromRequest retrieves a user ID from the request.
func getUserIDFromRequest(req *restful.Request) (string, error) {
	rawValue := req.Attribute(attributeUserID)
	if rawValue == nil {
		return "", fmt.Errorf("attribute missing: %s", attributeUserID)
	}
	userID, ok := rawValue.(string)
	if !ok {
		return "", fmt.Errorf("attribute has incorrect type %T: %s", rawValue, attributeUserID)
	}
	return userID, nil
}

// filterAppendUserInformation adds the user information to the request attributes.
func (c Config) filterAppendUserInformation(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	defer chain.ProcessFilter(req, resp) // Proceed no matter what.

	ctx := req.Request.Context()

	// The authentication does not use the *user*'s database connection; it doesn't need any permissions.
	// In fact, this is *how* we even get the user's information in the first place.
	db := c.DB.Session(&gorm.Session{NewDB: true})

	// Set the database in the request attributes.
	// If the user is not authenticated, then the database will be the main database with no CTEs applied.
	setDatabaseForRequest(req, db)

	var user *userInformation
	{
		var tokenString string

		// Load the token string from the Authorization header.
		{
			slog.DebugContext(ctx, "Authenticating with API token from the header.")
			authorization := req.Request.Header.Get("Authorization")
			slog.DebugContext(ctx, fmt.Sprintf("Authorization header: %s", authorization))
			if strings.HasPrefix(authorization, "Bearer ") {
				tokenString = strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
			}
		}

		if tokenString != "" {
			slog.DebugContext(ctx, fmt.Sprintf("Found an API token: %s", tokenString))
			var err error
			user, err = c.findUserInformationFromToken(db, tokenString)
			if err != nil {
				slog.WarnContext(ctx, fmt.Sprintf("Could not find user from token: %v", err))
			}
		}
	}

	// If we didn't end up with a user, then we're done.
	if user == nil {
		slog.DebugContext(ctx, "We were not able to authenticate the user.")

		req.SetAttribute(attributeUserID, "")
		req.SetAttribute(attributeUserIsSystemAdmin, false)
		return
	}

	slog.DebugContext(ctx, fmt.Sprintf("Authenticated user: %+v", *user))

	req.SetAttribute(attributeUserID, user.ID)
	req.SetAttribute(attributeUserIsSystemAdmin, user.SystemAdmin)

	// Now that we have the user information, we can set the database in the request attributes.
	setDatabaseForRequest(req, db)
}

type userInformation struct {
	ID          string
	SystemAdmin bool
}

func (c Config) findUserInformationFromToken(db *gorm.DB, tokenString string) (*userInformation, error) {
	if c.SystemToken != "" && tokenString == c.SystemToken {
		return &userInformation{
			ID:          "0",
			SystemAdmin: true,
		}, nil
	}

	claims, err := c.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Validate the user.
	var users []*schema.User
	err = db.Session(&gorm.Session{NewDB: true}).
		Where("username = ?", claims.Email).
		First(&users).
		Error
	if err != nil {
		return nil, fmt.Errorf("could not query for user: %w", err)
	}
	if len(users) == 0 {
		return nil, nil
	}
	user := users[0]

	return &userInformation{
		ID:          fmt.Sprintf("%d", user.ID),
		SystemAdmin: false,
	}, nil
}

// doRequireAuthentication requires authentication.
func doRequireAuthentication(requireAuthentication bool) func(routeBuilder *restful.RouteBuilder) {
	return func(routeBuilder *restful.RouteBuilder) {
		routeBuilder.Param(restful.HeaderParameter("Authorization", `This endpoint requires authentication, to be specified as a Bearer token, as "Bearer <token>", or as a Basic token, as "Basic <base64(username:password)>".`))
		routeBuilder.Returns(http.StatusUnauthorized, "Unauthorized", nil)
		routeBuilder.Metadata(MetadataAuthBasic, true)  // Add auth metadata for the OpenAPI docs.
		routeBuilder.Metadata(MetadataAuthBearer, true) // Add auth metadata for the OpenAPI docs.
		routeBuilder.Filter(filterRequireAuthentication)
	}
}

// filterRequireAuthentication requires authentication.
func filterRequireAuthentication(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	ctx := req.Request.Context()

	userID, err := getUserIDFromRequest(req)
	if err != nil {
		wrappedError := wrappedError{
			err: fmt.Errorf("could not get user ID from request: %w", err),
		}
		wrappedError.WriteError(resp)
		return
	}
	if userID == "" {
		slog.InfoContext(ctx, "User ID not found in request; this request is not authenticated.")
		wrappedError := wrappedError{
			err: fmt.Errorf("%w", httperror.ErrStatusUnauthorized),
		}
		wrappedError.WriteError(resp)
		return
	}
	slog.DebugContext(ctx, fmt.Sprintf("Required authentication successful for user: %s", userID))
	chain.ProcessFilter(req, resp)
}
