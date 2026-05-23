package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/downballot/downballot/downballotapi"
	"github.com/downballot/downballot/internal/api/downballotwrapper"
	"github.com/downballot/downballot/internal/apitoken"
	"github.com/downballot/downballot/internal/durationparser"
	"github.com/downballot/downballot/internal/schema"
	"github.com/downballot/downballot/internal/schema/sqltype"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/tekkamanendless/go-mailer"
	"github.com/threatmate/restfulwrapper"
	"gorm.io/gorm"
)

const TOTPPeriod = 300           // 5 minutes.
const TOTPDigits = otp.DigitsSix // 6 digits.
const TOTPSkew = 1               // Allow for a one-time password to be used up to 1 time period after it was generated.

// Email does not require authentication, since it is what creates authentication.
type PostAuthenticationEmailMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.MayHaveAuthenticatedUser
	downballotwrapper.UseDatabase
	_    string                     `api:"httppath:/authentication/email"`
	_    string                     `api:"doc" description:"Send a one-time password to the user's email address."`
	_    string                     `api:"notes" description:"This attempts to send a one-time password to the user's email address, if that e-mail address is associated with an account."`
	Body downballotapi.EmailRequest `api:"body"`
}

func (a *API) PostAuthenticationEmail(ctx context.Context, meta PostAuthenticationEmailMetadata) (output downballotapi.Envelope[downballotapi.EmailResponse], err error) {
	slog.InfoContext(ctx, fmt.Sprintf("Email: %s", meta.Body.Email))

	var users []*schema.User
	err = meta.DB.Session(&gorm.Session{}).
		Where("username = ?", meta.Body.Email).
		Find(&users).
		Error
	if err != nil {
		return output, err
	}

	if len(users) == 0 {
		// This isn't a valid user.

		// Don't do anything different; send the same message.
	} else {
		// This is a valid user.
		user := users[0]

		var userTOTP *schema.UserTOTP
		{
			var userTOTPs []*schema.UserTOTP
			err = meta.DB.Session(&gorm.Session{}).
				Where("user_id = ?", user.ID).
				Find(&userTOTPs).
				Error
			if err != nil {
				return output, err
			}
			if len(userTOTPs) == 0 {
				// This user doesn't have a TOTP yet.
				// Create one.

				key, err := totp.Generate(totp.GenerateOpts{
					Issuer:      "-",
					AccountName: "-",
					Period:      uint(TOTPPeriod),
					Digits:      otp.Digits(TOTPDigits),
					Algorithm:   otp.AlgorithmSHA1,
				})
				if err != nil {
					return output, fmt.Errorf("could not generate TOTP key: %w", err)
				}

				newTOTP := schema.UserTOTP{
					UserID: user.ID,
					Secret: sqltype.EncryptedString(key.Secret()),
				}
				err = meta.DB.Transaction(func(tx *gorm.DB) error {
					err = tx.Session(&gorm.Session{NewDB: true}).
						Create(&newTOTP).
						Error
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					return output, err
				}
				userTOTP = &newTOTP
			} else {
				// This user has a TOTP.
				userTOTP = userTOTPs[0]
			}
		}

		var oneTimePassword string
		{
			code, err := totp.GenerateCodeCustom(string(userTOTP.Secret), time.Now(), totp.ValidateOpts{
				Period:    uint(TOTPPeriod),
				Skew:      TOTPSkew,
				Digits:    otp.Digits(TOTPDigits),
				Algorithm: otp.AlgorithmSHA1,
			})
			if err != nil {
				return output, fmt.Errorf("could not generate TOTP code: %w", err)
			}
			oneTimePassword = code
		}

		slog.InfoContext(ctx, fmt.Sprintf("One-time password: %s", oneTimePassword))

		// Send the e-mail.
		if a.mailer == nil {
			slog.WarnContext(ctx, "No mailer configured; could not send e-mail.")
		} else {
			err = a.mailer.SendMail(ctx, mailer.Message{
				From: mail.Address{
					Name:    "Downballot",
					Address: "noreply@app.downballot.io",
				},
				To: mail.Address{
					Address: meta.Body.Email,
				},
				Subject:       "Your Downballot one-time password",
				BodyPlainText: fmt.Sprintf("Your one-time password is:\n\n%s", oneTimePassword),
			})
		}
	}

	output.Message = "OK"
	output.Success = true
	output.Data.Message = "Check your email for a one-time password."
	return output, nil
}

// Login does not require authentication, since it is what creates authentication.
// Note, however, that if you hit this endpoint with a token, you will essentially use that token as your credentials and receive a new token.
type PostAuthenticationLoginMetadata struct {
	restfulwrapper.HTTPMethodPOST
	downballotwrapper.MayHaveAuthenticatedUser
	downballotwrapper.UseDatabase
	_        string                     `api:"httppath:/authentication/login"`
	_        string                     `api:"doc" description:"Log in."`
	_        string                     `api:"notes" description:"This attempts to log in the user with a username and password.  Upon completion, this will provide the user with an API token that can be used in subsequent calls."`
	Body     downballotapi.LoginRequest `api:"body"`
	Lifetime *string                    `api:"query:lifetime"`
}

func (a *API) PostAuthenticationLogin(ctx context.Context, meta PostAuthenticationLoginMetadata) (output downballotapi.Envelope[downballotapi.LoginResponse], err error) {
	slog.InfoContext(ctx, fmt.Sprintf("Username: %s", meta.Body.Username))
	if meta.Body.Password == "" {
		slog.InfoContext(ctx, "Password: n/a")
	} else {
		slog.InfoContext(ctx, "Password: ********")
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
	slog.InfoContext(ctx, fmt.Sprintf("Expiration date: %v", claims.ExpiresAt))

	if meta.CurrentUser != nil {
		claims.Subject = meta.CurrentUser.EmailAddress
		claims.SessionIdentifier = meta.CurrentUser.ID

		slog.InfoContext(ctx, fmt.Sprintf("This request is already authenticated as: %s", claims.Subject))
	} else if meta.Body.Username != "" && meta.Body.Password != "" {
		var users []schema.User
		err = meta.DB.Session(&gorm.Session{}).
			Where("username = ?", meta.Body.Username).
			Find(&users).
			Error
		if err != nil {
			return output, err
		}

		if len(users) == 0 {
			return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "Invalid username or password")
		}

		if len(users) > 1 {
			return output, restfulwrapper.NewAPIResponseError(http.StatusInternalServerError, "Multiple users found with the same username")
		}

		user := users[0]

		var userTOTP *schema.UserTOTP
		{
			var userTOTPs []*schema.UserTOTP
			err = meta.DB.Session(&gorm.Session{}).
				Where("user_id = ?", user.ID).
				Find(&userTOTPs).
				Error
			if err != nil {
				return output, err
			}
			if len(userTOTPs) == 0 {
				return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "User does not have a TOTP")
			}
			userTOTP = userTOTPs[0]
		}

		// Verify the password.
		{
			var oneTimePassword string
			{
				code, err := totp.GenerateCodeCustom(string(userTOTP.Secret), time.Now(), totp.ValidateOpts{
					Period:    uint(TOTPPeriod),
					Skew:      TOTPSkew,
					Digits:    otp.Digits(TOTPDigits),
					Algorithm: otp.AlgorithmSHA1,
				})
				if err != nil {
					return output, fmt.Errorf("could not generate TOTP code: %w", err)
				}
				oneTimePassword = code
			}

			if oneTimePassword != meta.Body.Password {
				return output, restfulwrapper.NewAPIResponseError(http.StatusUnauthorized, "Invalid password")
			}
		}

		claims.Subject = meta.Body.Username
		claims.SessionIdentifier = user.SessionIdentifier
	} else {
		slog.InfoContext(ctx, "This request is not authenticated.")
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

	output.Message = "OK"
	output.Success = true
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
	slog.InfoContext(ctx, fmt.Sprintf("Username: %s", meta.Body.Username))

	// TODO: ATTEMPT TO RESET THE PASSWORD

	output.Message = "OK"
	output.Success = true
	output.Data.Email = meta.Body.Username
	return output, nil
}

// Status does not require authentication for historical reasons.  If no user is logged in, then the "user" field will be null.
type GetAuthenticationStatusMetadata struct {
	restfulwrapper.HTTPMethodGET
	downballotwrapper.MayHaveAuthenticatedUser
	_ string `api:"httppath:/authentication/status"`
	_ string `api:"doc" description:"Status."`
	_ string `api:"notes" description:"This checks the validity of the user's token."`
}

func (a *API) GetAuthenticationStatus(ctx context.Context, meta GetAuthenticationStatusMetadata) (output downballotapi.Envelope[downballotapi.AuthenticationStatusResponse], err error) {
	if meta.CurrentUser != nil {
		output.Message = "OK"
		output.Success = true
		output.Data.User = &downballotapi.AuthenticationStatusUser{
			ID:    fmt.Sprintf("%d", meta.CurrentUser.ID),
			Email: meta.CurrentUser.EmailAddress,
			Name:  meta.CurrentUser.Name,
			Admin: meta.CurrentUser.SystemAdmin,
		}
	} else {
		output.Message = "Not authenticated"
		output.Success = false
	}

	return output, nil
}
