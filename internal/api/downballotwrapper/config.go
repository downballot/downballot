package downballotwrapper

import (
	"crypto/rsa"

	"github.com/emicklei/go-restful/v3"
	"gorm.io/gorm"
)

// Config is the configuration for the middleware.
type Config struct {
	DB            *gorm.DB
	JWTPrivateKey *rsa.PrivateKey // This is the JWT private key, if any.
	JWTPublicKey  *rsa.PublicKey  // This is the JWT public key, if any.
	JWTSecret     []byte
	SystemToken   string
}

// Attributes returns the attributes for the middleware.
//
// This should be passed to the restfulwrapper.WebService.Attributes() method.
func (c *Config) Attributes() map[string]any {
	return map[string]any{}
}

// Do returns the middleware function.
//
// This should be passed to the restfulwrapper.WebService.Do() method.
func (c *Config) Do() func(routeBuilder *restful.RouteBuilder) {
	return func(routeBuilder *restful.RouteBuilder) {
		routeBuilder.Filter(c.filterAppendUserInformation)
	}
}
