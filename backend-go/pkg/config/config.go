package config

import (
	"os"
)

// Config holds all environment configuration for the application.
// All values are loaded from environment variables with sensible defaults
// for local development matching docker-compose.yml settings.
type Config struct {
	// Core
	DatabaseURL    string
	RedisURL       string
	RabbitMQURL    string
	JWTSecret      string
	ServerPort     string
	AppBaseURL     string

	// Object Storage (MinIO local / GCS prod)
	StorageEndpoint  string
	StorageAccessKey string
	StorageSecretKey string
	StorageBucket    string
	StorageUseSSL    bool

	// GCP Placeholders (populated in production)
	GCSProjectID        string
	GCSBucket           string
	GCSCredentialsJSON  string
	CloudSQLConnection  string
	FirebaseCredentials string
	TwilioAccountSID    string
	TwilioAuthToken     string
	TwilioPhoneNumber   string
	GeminiAPIKey        string
}

// Load reads all configuration from environment variables.
// Defaults match the local docker-compose setup so the app works
// out-of-the-box with `make up && go run ./cmd/api/`.
func Load() *Config {
	return &Config{
		// Core
		DatabaseURL: getEnv("DATABASE_URL", "postgres://rapid_user:rapid_password@localhost:5433/rapid_db?sslmode=disable"),
		RedisURL:    getEnv("REDIS_URL", "localhost:6379"),
		RabbitMQURL: getEnv("RABBITMQ_URL", "amqp://rapid_user:rapid_password@localhost:5673/"),
		JWTSecret:   getEnv("JWT_SECRET", "super-secret-hackathon-key"),
		ServerPort:  getEnv("SERVER_PORT", "8080"),
		AppBaseURL:  getEnv("APP_BASE_URL", "http://localhost:8080"),

		// Object Storage
		StorageEndpoint:  getEnv("STORAGE_ENDPOINT", "localhost:9002"),
		StorageAccessKey: getEnv("STORAGE_ACCESS_KEY", "rapid_admin"),
		StorageSecretKey: getEnv("STORAGE_SECRET_KEY", "rapid_password_123"),
		StorageBucket:    getEnv("STORAGE_BUCKET", "rapid-incidents"),
		StorageUseSSL:    false, // ASSUMPTION: local dev always uses HTTP

		// GCP (all empty by default — populated via Cloud Run env vars in production)
		GCSProjectID:        getEnv("GCS_PROJECT_ID", ""),
		GCSBucket:           getEnv("GCS_BUCKET_NAME", ""),
		GCSCredentialsJSON:  getEnv("GCS_CREDENTIALS_JSON", ""),
		CloudSQLConnection:  getEnv("CLOUD_SQL_CONNECTION_NAME", ""),
		FirebaseCredentials: getEnv("FIREBASE_CREDENTIALS_JSON", ""),
		TwilioAccountSID:    getEnv("TWILIO_ACCOUNT_SID", ""),
		TwilioAuthToken:     getEnv("TWILIO_AUTH_TOKEN", ""),
		TwilioPhoneNumber:   getEnv("TWILIO_PHONE_NUMBER", ""),
		GeminiAPIKey:        getEnv("GEMINI_API_KEY", ""),
	}
}

// getEnv reads an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists && value != "" {
		return value
	}
	return defaultValue
}
