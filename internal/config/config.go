package config

import (
	"fmt"
	"os"
)

// Config holds the application configuration
type Config struct {
	UseRealData      bool
	AlphaVantageKey  string
	JSONAddr         string
	GRPCAddr         string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() *Config {
	return &Config{
		UseRealData:      os.Getenv("USE_REAL_DATA") == "true",
		AlphaVantageKey:  getEnvWithDefault("ALPHA_VANTAGE_API_KEY", "demo"),
		JSONAddr:         getEnvWithDefault("JSON_ADDR", ":8080"),
		GRPCAddr:         getEnvWithDefault("GRPC_ADDR", ":8081"),
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.UseRealData && c.AlphaVantageKey == "" {
		return fmt.Errorf("ALPHA_VANTAGE_API_KEY is required when USE_REAL_DATA=true")
	}
	return nil
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}