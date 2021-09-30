package appconfig

// Config is the configuration for the API.
type Config struct {
	JWTSecret     string `json:"jwt_secret"`
	JWTPublicKey  string `json:"jwt_public_key"`
	JWTPrivateKey string `json:"jwt_private_key"`
}
