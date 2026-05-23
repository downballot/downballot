package apitoken

import (
	"fmt"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
)

// TokenClaims is the collection of claims that will be used in an authentication token.
type TokenClaims struct {
	jwt.StandardClaims // These are the standard JWT claims.

	// Custom claims go here.
	SessionIdentifier uint64 `json:"session_identifier"`
}

// Valid returns an error of the claims are invalid (or nil otherwise).
func (c TokenClaims) Valid() error {
	if c.ExpiresAt > 0 && c.ExpiresAt < time.Now().Unix() {
		return fmt.Errorf("token is expired (%s)", time.Unix(c.ExpiresAt, 0).Format(time.RFC3339))
	}
	return nil
}
