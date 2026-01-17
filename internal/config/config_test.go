package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name             string
		envVars          map[string]string
		expectedUseReal  bool
		expectedAPIKey   string
		expectedJSONAddr string
		expectedGRPCAddr string
	}{
		{
			name: "Default configuration",
			envVars: map[string]string{},
			expectedUseReal:  false,
			expectedAPIKey:   "demo",
			expectedJSONAddr: ":8080",
			expectedGRPCAddr: ":8081",
		},
		{
			name: "Custom configuration",
			envVars: map[string]string{
				"USE_REAL_DATA":          "true",
				"ALPHA_VANTAGE_API_KEY":  "test-key",
				"JSON_ADDR":              ":9000",
				"GRPC_ADDR":              ":9001",
			},
			expectedUseReal:  true,
			expectedAPIKey:   "test-key",
			expectedJSONAddr: ":9000",
			expectedGRPCAddr: ":9001",
		},
		{
			name: "Partial configuration",
			envVars: map[string]string{
				"USE_REAL_DATA": "true",
			},
			expectedUseReal:  true,
			expectedAPIKey:   "demo",
			expectedJSONAddr: ":8080",
			expectedGRPCAddr: ":8081",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment variables
			origUseRealData := os.Getenv("USE_REAL_DATA")
			origAPIKey := os.Getenv("ALPHA_VANTAGE_API_KEY")
			origJSONAddr := os.Getenv("JSON_ADDR")
			origGRPCAddr := os.Getenv("GRPC_ADDR")

			// Unset all environment variables first
			os.Unsetenv("USE_REAL_DATA")
			os.Unsetenv("ALPHA_VANTAGE_API_KEY")
			os.Unsetenv("JSON_ADDR")
			os.Unsetenv("GRPC_ADDR")

			// Set environment variables for this test
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Load config
			cfg := LoadConfig()

			// Restore original environment variables
			os.Setenv("USE_REAL_DATA", origUseRealData)
			os.Setenv("ALPHA_VANTAGE_API_KEY", origAPIKey)
			os.Setenv("JSON_ADDR", origJSONAddr)
			os.Setenv("GRPC_ADDR", origGRPCAddr)

			// Verify config
			if cfg.UseRealData != tt.expectedUseReal {
				t.Errorf("LoadConfig().UseRealData = %v, want %v", cfg.UseRealData, tt.expectedUseReal)
			}

			if cfg.AlphaVantageKey != tt.expectedAPIKey {
				t.Errorf("LoadConfig().AlphaVantageKey = %v, want %v", cfg.AlphaVantageKey, tt.expectedAPIKey)
			}

			if cfg.JSONAddr != tt.expectedJSONAddr {
				t.Errorf("LoadConfig().JSONAddr = %v, want %v", cfg.JSONAddr, tt.expectedJSONAddr)
			}

			if cfg.GRPCAddr != tt.expectedGRPCAddr {
				t.Errorf("LoadConfig().GRPCAddr = %v, want %v", cfg.GRPCAddr, tt.expectedGRPCAddr)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid mock config",
			config: &Config{
				UseRealData:      false,
				AlphaVantageKey:  "",
				JSONAddr:         ":8080",
				GRPCAddr:         ":8081",
			},
			wantErr: false,
		},
		{
			name: "Valid real data config",
			config: &Config{
				UseRealData:      true,
				AlphaVantageKey:  "test-key",
				JSONAddr:         ":8080",
				GRPCAddr:         ":8081",
			},
			wantErr: false,
		},
		{
			name: "Invalid real data config - missing API key",
			config: &Config{
				UseRealData:      true,
				AlphaVantageKey:  "",
				JSONAddr:         ":8080",
				GRPCAddr:         ":8081",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetEnvWithDefault(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		want         string
	}{
		{
			name:         "Environment variable set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "custom",
			want:         "custom",
		},
		{
			name:         "Environment variable not set",
			key:          "TEST_VAR",
			defaultValue: "default",
			envValue:     "",
			want:         "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			got := getEnvWithDefault(tt.key, tt.defaultValue)
			if got != tt.want {
				t.Errorf("getEnvWithDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}