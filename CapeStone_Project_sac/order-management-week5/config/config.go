package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds application settings loaded from environment variables.
type Config struct {
	Environment       string
	AppPort           string
	DBHost            string
	DBPort            string
	DBName            string
	DBUser            string
	DBPassword        string
	DBSSLMode         string
	APIKey            string
	ProductServiceURL string
	ProductTimeout    time.Duration
	ProductRetries    int
}

func Load() (Config, error) {
	_ = godotenv.Load()

	cfg := Config{
		Environment:       envString("APP_ENV"),
		AppPort:           envString("APP_PORT"),
		DBHost:            envString("DB_HOST"),
		DBPort:            envString("DB_PORT"),
		DBName:            envString("DB_NAME"),
		DBUser:            envString("DB_USER"),
		DBPassword:        envString("DB_PASSWORD"),
		DBSSLMode:         envString("DB_SSLMODE"),
		APIKey:            envString("API_KEY"),
		ProductServiceURL: envString("PRODUCT_SERVICE_URL"),
		ProductTimeout:    time.Duration(envInt("PRODUCT_TIMEOUT_MS")) * time.Millisecond,
		ProductRetries:    envInt("PRODUCT_RETRIES"),
	}

	if cfg.Environment == "production" && cfg.DBHost == "" {
		return Config{}, fmt.Errorf("DB_HOST is required in production mode")
	}

	return cfg, nil
}

func envString(key string) string {
	if value, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func envInt(key string) int {
	if value, ok := os.LookupEnv(key); ok {
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return 0
}
