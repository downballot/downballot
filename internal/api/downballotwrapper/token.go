package downballotwrapper

import (
	"fmt"

	"github.com/dgrijalva/jwt-go"
	"github.com/downballot/downballot/internal/apitoken"
)

// ValidateToken validates an API token (from a string) and returns the token's claims,
// which can be used to learn more about the user.
//
// If the token has expired, then this fails with an error.
func (c Config) ValidateToken(tokenString string) (apitoken.TokenClaims, error) {
	var claims apitoken.TokenClaims
	_, err := jwt.ParseWithClaims(tokenString, &claims,
		func(t *jwt.Token) (any, error) {
			if c.JWTSecret != nil {
				return c.JWTSecret, nil
			}
			if c.JWTPublicKey != nil {
				return c.JWTPublicKey, nil
			}
			return jwt.UnsafeAllowNoneSignatureType, nil
		},
	)
	if err != nil {
		return claims, fmt.Errorf("could not parse token: %v", err)
	}

	return claims, nil
}
