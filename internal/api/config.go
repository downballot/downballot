package api

type Config struct {
	JWTSecret     string
	JWTPublicKey  string
	JWTPrivateKey string

	MasterToken            string // This is the master token for full system authentication.
	SendGridAPIKey         string // This is the SendGrid API key for sending emails.
	GoogleMapsServerAPIKey string // This is the Google Maps Server API key for geocoding.
}
