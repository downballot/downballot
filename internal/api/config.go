package api

type Config struct {
	JWTSecret     string
	JWTPublicKey  string
	JWTPrivateKey string

	MasterToken string
}
