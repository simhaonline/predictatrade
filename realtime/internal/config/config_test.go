package config

import (
	"os"
	"testing"
)

// P1-002: simulated provider mode must be rejected in production
func TestValidateRejectsSimulatedInProduction(t *testing.T) {
	os.Setenv("NODE_ENV", "production")
	defer os.Unsetenv("NODE_ENV")

	cfg := &Config{
		DBURL:        "postgres://user:securepass@db.example.com:5432/predictatrade",
		WSPort:       13081,
		ProviderMode: "simulated",
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for simulated provider in production")
	}
}

// P1-002: agent provider mode is allowed in production
func TestValidateAllowsAgentInProduction(t *testing.T) {
	os.Setenv("NODE_ENV", "production")
	defer os.Unsetenv("NODE_ENV")

	cfg := &Config{
		DBURL:        "postgres://user:securepass@db.example.com:5432/predictatrade",
		WSPort:       13081,
		ProviderMode: "agent",
	}
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for agent provider in production, got: %v", err)
	}
}

// P1-002: simulated provider mode is allowed in development
func TestValidateAllowsSimulatedInDevelopment(t *testing.T) {
	os.Unsetenv("NODE_ENV")
	os.Unsetenv("APP_ENV")

	cfg := &Config{
		DBURL:        "postgres://user:pass@localhost:5432/predictatrade",
		WSPort:       13081,
		ProviderMode: "simulated",
	}
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for simulated provider in development, got: %v", err)
	}
}

// P2-002: insecure DB password must be rejected in production
func TestValidateRejectsInsecureDBPasswordInProduction(t *testing.T) {
	os.Setenv("NODE_ENV", "production")
	defer os.Unsetenv("NODE_ENV")

	cfg := &Config{
		DBURL:        "postgres://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade",
		WSPort:       13081,
		ProviderMode: "agent",
	}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for insecure DB password in production")
	}
}

// P2-002: secure DB URL is allowed in production
func TestValidateAllowsSecureDBInProduction(t *testing.T) {
	os.Setenv("NODE_ENV", "production")
	defer os.Unsetenv("NODE_ENV")

	cfg := &Config{
		DBURL:        "postgres://app_user:SecureR4nd0mP@ss!@db.internal:5432/predictatrade",
		WSPort:       13081,
		ProviderMode: "agent",
	}
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for secure DB URL in production, got: %v", err)
	}
}

// P2-002: insecure DB password is allowed in development
func TestValidateAllowsInsecureDBInDevelopment(t *testing.T) {
	os.Unsetenv("NODE_ENV")
	os.Unsetenv("APP_ENV")

	cfg := &Config{
		DBURL:        "postgres://pat_admin:pat_local_dev_only@127.0.0.1:5432/predictatrade",
		WSPort:       13081,
		ProviderMode: "agent",
	}
	err := cfg.Validate()
	if err != nil {
		t.Errorf("Expected no error for insecure DB password in development, got: %v", err)
	}
}

// Basic validation
func TestValidateRequiresDBURL(t *testing.T) {
	cfg := &Config{DBURL: "", WSPort: 13081}
	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for missing DATABASE_URL")
	}
}

func TestIsInsecureSecretDetection(t *testing.T) {
	insecure := []string{"", "CHANGE_ME_IN_PRODUCTION", "changeme", "secret", "placeholder", "development"}
	for _, s := range insecure {
		if !IsInsecureSecret(s) {
			t.Errorf("Expected '%s' to be detected as insecure", s)
		}
	}
	secure := "a-very-long-and-random-production-secret-key-1234567890"
	if IsInsecureSecret(secure) {
		t.Error("Expected secure secret to NOT be detected as insecure")
	}
}
