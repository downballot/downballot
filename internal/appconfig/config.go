package appconfig

// Config is the configuration for the API.
type Config struct {
	DatabaseDriver string `json:"database_driver"`
	DatabaseString string `json:"database_string"`

	JWTSecret     string `json:"jwt_secret"`
	JWTPublicKey  string `json:"jwt_public_key"`
	JWTPrivateKey string `json:"jwt_private_key"`

	MasterToken string `json:"master_token"`
}
