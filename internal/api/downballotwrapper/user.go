package downballotwrapper

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/WinterYukky/gorm-extra-clause-plugin/exclause"
	"github.com/downballot/downballot/iam"
	"github.com/downballot/downballot/internal/schema"
	"github.com/downballot/downballot/permissionset"
	"github.com/emicklei/go-restful/v3"
	"github.com/tekkamanendless/httperror"
	"gorm.io/gorm"
)

// User is a user in the system.
type User struct {
	ID                             uint64                                 // The user ID.  This will be "0" if the system token is used.
	EmailAddress                   string                                 // The user's email address.  This will be "@system" if the system token is used.
	Name                           string                                 // The user's name.  This will be "System User" if the system token is used.
	SystemAdmin                    bool                                   // Whether the user is a system administrator.  This is only true if the system token is used.
	organizationToPermissionSetMap map[uint64]permissionset.PermissionSet // The user's permission set for each organization.
}

// PermissionSetForOrganization returns the user's permission set for an organization.
func (u *User) PermissionSetForOrganization(organizationID uint64) permissionset.PermissionSet {
	if u.SystemAdmin {
		return *permissionset.NewPermissionSet(iam.Permissions...)
	}
	if permissionSet, ok := u.organizationToPermissionSetMap[organizationID]; ok {
		return permissionSet
	}
	return *permissionset.NewPermissionSet()
}

// RequireAuthenticatedUser requires an authenticated user.
type RequireAuthenticatedUser struct {
	CurrentUser User `api:"downballot.currentUser"`
}

// MayHaveAuthenticatedUser may have an authenticated user.
type MayHaveAuthenticatedUser struct {
	CurrentUser *User `api:"downballot.currentUser"`
}

// getUserFromRequest retrieves a user ID from the request.
func getUserFromRequest(req *restful.Request) (*User, error) {
	rawValue := req.Attribute(attributeUser)
	if rawValue == nil {
		return nil, fmt.Errorf("attribute missing: %s", attributeUser)
	}
	user, ok := rawValue.(*User)
	if !ok {
		return nil, fmt.Errorf("attribute has incorrect type %T: %s", rawValue, attributeUser)
	}
	return user, nil
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

	var user *User
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

	if user == nil {
		slog.DebugContext(ctx, "We were not able to authenticate the user.")
	} else {
		slog.DebugContext(ctx, fmt.Sprintf("Authenticated user: %+v", *user))

		// Mask the "organization" table.
		{
			// This is the name of the real "organization" table.
			//
			// MySQL is smart enough to know that we're not referring to a table that we haven't created yet, but SQLite is not.
			//
			// SQLite will fail with this error: SQL logic error: circular reference: organization (1)
			// So, to work around that, we're going to insert the schema name, which is "main", so that SQLite doesn't get confused.
			organizationTableName := schema.Organization{}.TableName()
			originalOrganizationTableName := organizationTableName
			switch db.Dialector.Name() {
			case "sqlite":
				originalOrganizationTableName = "main." + organizationTableName
			}

			subQuery := db.Session(&gorm.Session{NewDB: true, Initialized: true}).
				Select(strings.Join([]string{
					"organization.id",
					"organization.name",
				}, ", ")).
				Table(originalOrganizationTableName).
				InnerJoins("INNER JOIN user_organization_map ON user_organization_map.organization_id = organization.id")
			if user.SystemAdmin {
				// Don't do anything else.
			} else {
				subQuery = subQuery.Where("user_organization_map.user_id = ?", user.ID)
			}

			withClause := exclause.With{
				Recursive: false,
				CTEs: []exclause.CTE{
					{
						Name: organizationTableName,
						Columns: []string{
							"id",
							"name",
						},
						Subquery: exclause.Subquery{
							DB: subQuery,
						},
					},
				},
			}

			db = db.Clauses(withClause)
		}
	}

	req.SetAttribute(attributeUser, user)

	// Now that we have the user information, we can set the database in the request attributes.
	setDatabaseForRequest(req, db)
}

func (c Config) findUserInformationFromToken(db *gorm.DB, tokenString string) (*User, error) {
	if c.SystemToken != "" && tokenString == c.SystemToken {
		return &User{
			ID:           0,
			EmailAddress: "@system",
			Name:         "System User",
			SystemAdmin:  true,
		}, nil
	}

	claims, err := c.ValidateToken(tokenString)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Validate the user.
	var users []*schema.User
	err = db.Session(&gorm.Session{}).
		Where("username = ?", claims.Subject).
		Where("session_identifier = ?", claims.SessionIdentifier).
		First(&users).
		Error
	if err != nil {
		return nil, fmt.Errorf("could not query for user: %w", err)
	}
	if len(users) == 0 {
		return nil, nil
	}
	user := users[0]

	organizationToPermissionSetMap := map[uint64]permissionset.PermissionSet{}
	{
		var userOrganizationMaps []*schema.UserOrganizationMap
		err = db.Session(&gorm.Session{}).
			Where("user_id = ?", user.ID).
			Find(&userOrganizationMaps).
			Error
		if err != nil {
			return nil, fmt.Errorf("could not query for user group maps: %w", err)
		}
		for _, userOrganizationMap := range userOrganizationMaps {
			permissionSet := permissionset.PermissionSet{}
			if userOrganizationMap.Owner {
				permissionSet.AddPermission(iam.Permissions...)
			} else {
				permissionSet.AddPermission(permissionset.Permission(iam.IAMFilterRead))
				permissionSet.AddPermission(permissionset.Permission(iam.IAMGroupRead))
				permissionSet.AddPermission(permissionset.Permission(iam.IAMPersonRead))
				permissionSet.AddPermission(permissionset.Permission(iam.IAMPersonFieldDefinitionRead))
			}
			organizationToPermissionSetMap[userOrganizationMap.OrganizationID] = permissionSet
		}
	}

	return &User{
		ID:                             user.ID,
		EmailAddress:                   user.Username,
		Name:                           user.Name,
		SystemAdmin:                    false,
		organizationToPermissionSetMap: organizationToPermissionSetMap,
	}, nil
}

// doRequireAuthentication requires authentication.
func doRequireAuthentication(requireAuthentication bool) func(routeBuilder *restful.RouteBuilder) {
	return func(routeBuilder *restful.RouteBuilder) {
		routeBuilder.Param(restful.HeaderParameter("Authorization", `This endpoint requires authentication, to be specified as a Bearer token, as "Bearer <token>", or as a Basic token, as "Basic <base64(username:password)>".`))
		routeBuilder.Returns(http.StatusUnauthorized, "Unauthorized", nil)
		routeBuilder.Metadata(MetadataAuthBasic, true)  // Add auth metadata for the OpenAPI docs.
		routeBuilder.Metadata(MetadataAuthBearer, true) // Add auth metadata for the OpenAPI docs.
		routeBuilder.Filter(filterRequireAuthentication(requireAuthentication))
	}
}

// filterRequireAuthentication requires authentication.
func filterRequireAuthentication(requireAuthentication bool) func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		ctx := req.Request.Context()

		user, err := getUserFromRequest(req)
		if err != nil {
			wrappedError := wrappedError{
				err: fmt.Errorf("could not get user ID from request: %w", err),
			}
			wrappedError.WriteError(resp)
			return
		}

		if requireAuthentication {
			if user == nil {
				slog.InfoContext(ctx, "User ID not found in request; this request is not authenticated.")
				wrappedError := wrappedError{
					err: fmt.Errorf("%w", httperror.ErrStatusUnauthorized),
				}
				wrappedError.WriteError(resp)
				return
			}
			slog.DebugContext(ctx, fmt.Sprintf("Required authentication successful for user: %+v", user))
		}
		chain.ProcessFilter(req, resp)
	}
}
