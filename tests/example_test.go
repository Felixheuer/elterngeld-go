package tests

import (
	"testing"

	"elterngeld-portal/config"
	"elterngeld-portal/pkg/auth"
)

func TestJWTService(t *testing.T) {
	// Load test configuration
	cfg := &config.Config{
		JWT: config.JWTConfig{
			Secret:        "test-secret",
			AccessExpiry:  config.ParseDuration("15m"),
			RefreshExpiry: config.ParseDuration("168h"),
		},
	}

	jwtService := auth.NewJWTService(cfg)

	if jwtService == nil {
		t.Error("JWT service should not be nil")
	}
}

func TestConfigValidation(t *testing.T) {
	// Test that configuration loading works
	// This is a placeholder test
	t.Log("Configuration test placeholder")
}
