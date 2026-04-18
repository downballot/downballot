package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/apitoken"
	"github.com/downballot/downballot/internal/durationparser"
	"github.com/sirupsen/logrus"
	"github.com/threatmate/restfulwrapper"
)

// Create-account does not accept authentication, since this is what makes an account in the first place.
type PostAuthenticationCreateAccountMetadata struct {
	restfulwrapper.HTTPMethodPOST
	_    string                             `api:"httppath:/authentication/create-account"`
	_    string                             `api:"doc" description:"Create a new account."`
	_    string                             `api:"notes" description:"This attempts to create a new account."`
	Body downballotapi.CreateAccountRequest `api:"body"`
}

func (a *API) PostAuthenticationCreateAccount(ctx context.Context, meta PostAuthenticationCreateAccountMetadata) (output downballotapi.Envelope[downballotapi.CreateAccountResponse], err error) {
	logrus.WithContext(ctx).Infof("Username: %s", meta.Body.Username)
	logrus.WithContext(ctx).Infof("Password: ********")

	// TODO: CREATE THE USER

	return output, nil
}

// Login does not require authentication, since it is what creates authentication.
// Note, however, that if you hit this endpoint with a token, you will essentially use that token as your credentials and receive a new token.
type PostAuthenticationLoginMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.MayHaveAuthenticatedUser
	_        string                     `api:"httppath:/authentication/login"`
	_        string                     `api:"doc" description:"Log in."`
	_        string                     `api:"notes" description:"This attempts to log in the user with a username and password.  Upon completion, this will provide the user with an API token that can be used in subsequent calls."`
	Body     downballotapi.LoginRequest `api:"body:consumes:*/*;empty"`
	Lifetime *string                    `api:"query:lifetime"`
}

func (a *API) PostAuthenticationLogin(ctx context.Context, meta PostAuthenticationLoginMetadata) (output downballotapi.Envelope[downballotapi.LoginResponse], err error) {
	logrus.WithContext(ctx).Infof("Username: %s", meta.Body.Username)
	if meta.Body.Password == "" {
		logrus.WithContext(ctx).Infof("Password: n/a")
	} else {
		logrus.WithContext(ctx).Infof("Password: ********")
	}

	claims := apitoken.TokenClaims{}
	if meta.Lifetime != nil {
		expirationDate, err := durationparser.Parse(time.Now(), *meta.Lifetime)
		if err != nil {
			return output, restfulwrapper.NewAPIQueryParameterError("lifetime", fmt.Errorf("could not parse 'lifetime' value %q: %v", *meta.Lifetime, err))
		}
		if expirationDate != nil {
			claims.ExpiresAt = expirationDate.Unix()
		}
	}
	logrus.WithContext(ctx).Infof("Expiration date: %v", claims.ExpiresAt)

	if meta.CurrentUser != nil {
		claims.Subject = meta.CurrentUser.ID
		claims.Email = meta.CurrentUser.EmailAddress

		logrus.WithContext(ctx).Infof("This request is already authenticated as: %s", claims.Subject)
	} else if meta.Body.Username != "" && meta.Body.Password != "" {
		// TODO: SIGN THE USER IN.

		claims.Subject = meta.Body.Username
		claims.Email = meta.Body.Username
	} else {
		logrus.WithContext(ctx).Infof("This request is not authenticated.")
	}

	// If we couldn't login the user, then fail.
	if claims.Subject == "" {
		return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "")
	}

	// Generate a token for the user.

	var signingMethod jwt.SigningMethod
	var signingKey any

	if a.jwtSecret != nil {
		signingMethod = jwt.SigningMethodHS512
		signingKey = a.jwtSecret
	} else if a.jwtPrivateKey != nil {
		signingMethod = jwt.SigningMethodRS256
		signingKey = a.jwtPrivateKey
	} else {
		signingMethod = jwt.SigningMethodNone
		signingKey = jwt.UnsafeAllowNoneSignatureType
	}
	token := jwt.NewWithClaims(signingMethod, claims)
	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		return output, err
	}

	output.Data.UserID = meta.Body.Username
	output.Data.Token = tokenString
	return output, nil
}

// Reset-password does not accept authentication, since you're only resetting your password if you can't log in.
type PostAuthenticationResetPasswordMetadata struct {
	restfulwrapper.HTTPMethodPOST
	_    string                             `api:"httppath:/authentication/reset-password"`
	_    string                             `api:"doc" description:"Reset a user's password."`
	_    string                             `api:"notes" description:"This attempts to reset a user's password.  This will send the user an e-mail with a link to click on to reset her password."`
	Body downballotapi.ResetPasswordRequest `api:"body"`
}

func (a *API) PostAuthenticationResetPassword(ctx context.Context, meta PostAuthenticationResetPasswordMetadata) (output downballotapi.Envelope[downballotapi.ResetPasswordResponse], err error) {
	logrus.WithContext(ctx).Infof("Username: %s", meta.Body.Username)

	// TODO: ATTEMPT TO RESET THE PASSWORD

	output.Data.Email = meta.Body.Username
	return output, nil
}

// Status does not require authentication for historical reasons.  If no user is logged in, then the "user" field will be null.
type GetAuthenticationStatusMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.MayHaveAuthenticatedUser
	_    string                            `api:"httppath:/authentication/status"`
	_    string                            `api:"doc" description:"Status."`
	_    string                            `api:"notes" description:"This checks the validity of the user's token."`
	Body downballotapi.RegisterUserRequest `api:"body"`
}

func (a *API) GetAuthenticationStatus(ctx context.Context, meta GetAuthenticationStatusMetadata) (output downballotapi.Envelope[downballotapi.AuthenticationStatusResponse], err error) {
	if meta.CurrentUser != nil {
		output.Data.User = &downballotapi.AuthenticationStatusUser{
			ID:    meta.CurrentUser.ID,
			Email: meta.CurrentUser.EmailAddress,
			Name:  meta.CurrentUser.Name,
			Admin: meta.CurrentUser.SystemAdmin,
		}
	}

	return output, nil
}
