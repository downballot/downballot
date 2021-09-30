package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/apitoken"
	"github.com/downballot/downballot/internal/durationparser"
	restful "github.com/emicklei/go-restful/v3"
	"github.com/sirupsen/logrus"
)

func (i *Instance) registerAuthenticationEndpoints(ws *restful.WebService) {
	// Create-account does not accept authentication, since this is what makes an account in the first place.
	ws.Route(
		ws.POST("authentication/create-account").To(i.postAuthenticationCreateAccount).
			Doc(`Create a new account`).
			Notes(`This attempts to create a new account.`).
			Reads(downballotapi.CreateAccountRequest{}).
			Returns(http.StatusOK, "OK", downballotapi.CreateAccountResponse{}),
	)
	// Login does not require authentication, since it is what creates authentication.
	// Note, however, that if you hit this endpoint with a token, you will essentially use that token as your credentials and receive a new token.
	ws.Route(
		ws.POST("authentication/login").To(i.postAuthenticationLogin).
			Doc(`Log in`).
			Notes(`This attempts to log in the user with a username and password.  Upon completion, this will provide the user with an API token that can be used in subsequent calls.`).
			Do(i.doAcceptAuthentication).
			Param(restful.QueryParameter("lifetime", `The lifetime of the token.  This can be "forever" or a number followed by "s" for seconds, "m" for minutes, "h" for hours, "d" for days, "M" for months, "y" for years".`)).
			Reads(downballotapi.LoginRequest{}).
			Consumes(restful.MIME_JSON, "*/*"). // Allow empty bodies.
			Returns(http.StatusOK, "OK", downballotapi.LoginResponse{}),
	)
	// Reset-password does not accept authentication, since you're only resetting your password if you can't log in.
	ws.Route(
		ws.POST("authentication/reset-password").To(i.postAuthenticationResetPassword).
			Doc(`Reset a user's password`).
			Notes(`This attempts to reset a user's password.  This will send the user an e-mail with a link to click on to reset her password.`).
			Reads(downballotapi.ResetPasswordRequest{}).
			Returns(http.StatusOK, "OK", downballotapi.ResetPasswordResponse{}),
	)
	// Status does not require authentication for historical reasons.  If no user is logged in, then the "user" field will be null.
	ws.Route(
		ws.GET("authentication/status").To(i.getAuthenticationStatus).
			Doc(`Status`).
			Notes(`This checks the validity of the user's token.`).
			Do(i.doAcceptAuthentication).
			Returns(http.StatusOK, "OK", downballotapi.AuthenticationStatusResponse{}),
	)
}

func (i *Instance) postAuthenticationCreateAccount(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	var input downballotapi.CreateAccountRequest
	err := request.ReadEntity(&input)
	if err != nil {
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	logrus.WithContext(ctx).Infof("Username: %s", input.Username)
	logrus.WithContext(ctx).Infof("Password: ********")

	// TODO: CREATE THE USER

	output := downballotapi.CreateAccountResponse{
		//Email: user.EmailAddress,
	}
	WriteEntity(ctx, response, output)
}

func (i *Instance) postAuthenticationLogin(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	var input downballotapi.LoginRequest
	if request.Request.ContentLength > 0 {
		err := request.ReadEntity(&input)
		if err != nil {
			WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
			return
		}
	}

	logrus.WithContext(ctx).Infof("Username: %s", input.Username)
	if input.Password == "" {
		logrus.WithContext(ctx).Infof("Password: n/a")
	} else {
		logrus.WithContext(ctx).Infof("Password: ********")
	}

	claims := apitoken.TokenClaims{}
	if durationString := request.QueryParameter("lifetime"); durationString != "" {
		expirationDate, err := durationparser.Parse(time.Now(), durationString)
		if err != nil {
			WriteHeaderAndError(ctx, response, http.StatusBadRequest, fmt.Errorf("could not parse 'lifetime' value %q: %v", durationString, err))
			return
		}
		if expirationDate != nil {
			claims.ExpiresAt = expirationDate.Unix()
		}
	}
	logrus.WithContext(ctx).Infof("Expiration date: %v", claims.ExpiresAt)

	if request.Attribute(AttributeLoggedIn) == true {
		// TODO: Allow impersonation if the user is an admin.
		if request.Attribute(AttributeUserID) != nil {
			claims.Subject = request.Attribute(AttributeUserID).(string)
			claims.Email = request.Attribute(AttributeUserID).(string)
		} else {
			WriteHeaderAndText(ctx, response, http.StatusBadRequest, "User ID or username are not set")
			return
		}
		logrus.WithContext(ctx).Infof("This request is already authenticated as: %s", claims.Subject)
	} else if input.Username != "" && input.Password != "" {
		// TODO: SIGN THE USER IN.

		claims.Subject = input.Username
		claims.Email = input.Username
	} else {
		logrus.WithContext(ctx).Infof("This request is not authenticated.")
	}

	// If we couldn't login the user, then fail.
	if claims.Subject == "" {
		WriteHeaderAndText(ctx, response, http.StatusUnauthorized, "Not logged in")
		return
	}

	// Generate a token for the user.

	var signingMethod jwt.SigningMethod
	var signingKey interface{}

	if i.jwtSecret != nil {
		signingMethod = jwt.SigningMethodHS512
		signingKey = i.jwtSecret
	} else if i.jwtPrivateKey != nil {
		signingMethod = jwt.SigningMethodRS256
		signingKey = i.jwtPrivateKey
	} else {
		signingMethod = jwt.SigningMethodNone
		signingKey = jwt.UnsafeAllowNoneSignatureType
	}
	token := jwt.NewWithClaims(signingMethod, claims)
	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		logrus.WithContext(ctx).Errorf("Could not create the token string: %v", err)
		WriteHeaderAndError(ctx, response, http.StatusInternalServerError, err)
		return
	}

	output := downballotapi.LoginResponse{
		UserID: input.Username,
		Token:  tokenString,
	}
	WriteEntity(ctx, response, output)
}

func (i *Instance) postAuthenticationResetPassword(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	var input downballotapi.ResetPasswordRequest
	err := request.ReadEntity(&input)
	if err != nil {
		WriteHeaderAndError(ctx, response, http.StatusBadRequest, err)
		return
	}

	logrus.WithContext(ctx).Infof("Username: %s", input.Username)

	// TODO: ATTEMPT TO RESET THE PASSWORD

	output := downballotapi.ResetPasswordResponse{
		Email: input.Username,
	}
	WriteEntity(ctx, response, output)
}

func (i *Instance) getAuthenticationStatus(request *restful.Request, response *restful.Response) {
	ctx := request.Request.Context()

	output := downballotapi.AuthenticationStatusResponse{
		User: nil,
	}
	if request.Attribute(AttributeUserID) != nil {
		output.User = &downballotapi.AuthenticationStatusUser{
			ID:    request.Attribute(AttributeUserID).(string),
			Email: request.Attribute(AttributeUserID).(string),
			Name:  request.Attribute(AttributeUserName).(string),
			Admin: request.Attribute(AttributeSystemAdmin).(bool),
		}
	}

	WriteEntity(ctx, response, output)
}
