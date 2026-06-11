package apitoken

import (
	"fmt"
	"testing"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/stretchr/testify/assert"
)

func TestTokenClaims(t *testing.T) {
	rows := []struct {
		description string
		claims      TokenClaims
		valid       bool
	}{
		{
			description: "No ExpiriresAt is valid",
			claims: TokenClaims{
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: 0,
				},
			},
			valid: true,
		},
		{
			description: "Past ExpiriresAt is invalid",
			claims: TokenClaims{
				StandardClaims: jwt.StandardClaims{
					ExpiresAt: int64(time.Now().Add(-60 * time.Second).Unix()),
				},
			},
			valid: false,
		},
	}
	for rowIndex, row := range rows {
		t.Run(fmt.Sprintf("%d/%s", rowIndex, row.description), func(t *testing.T) {
			err := row.claims.Valid()
			if !row.valid {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
		})
	}
}
