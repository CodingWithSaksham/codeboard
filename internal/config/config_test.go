package config

import (
	"os"
	"testing"
)

// Tests that getOrDefault returns the env variable value
// when the variable is set in the environment.
func TestGetOrDefault_EnvSet(t *testing.T) {
	t.Setenv("TEST_KEY", "myvalue")
	got := getOrDefault("TEST_KEY", "fallback")
	if got != "myvalue" {
		t.Errorf("expected %q, got %q", "myvalue", got)
	}
}

// Tests that getOrDefault returns the fallback string
// when the environment variable is not set.
func TestGetOrDefault_EnvNotSet(t *testing.T) {
	os.Unsetenv("NONEXISTENT_KEY_XYZ")
	got := getOrDefault("NONEXISTENT_KEY_XYZ", "default_val")
	if got != "default_val" {
		t.Errorf("expected %q, got %q", "default_val", got)
	}
}

// Tests that getOrDefault returns the fallback when
// the environment variable is set to an empty string.
func TestGetOrDefault_EmptyEnvValue(t *testing.T) {
	t.Setenv("EMPTY_KEY", "")
	got := getOrDefault("EMPTY_KEY", "fallback")
	if got != "fallback" {
		t.Errorf("expected fallback %q, got %q", "fallback", got)
	}
}

// Tests that mustGet returns the correct value
// when a required environment variable is present.
func TestMustGet_Present(t *testing.T) {
	t.Setenv("SECRET_KEY", "supersecret")
	got := mustGet("SECRET_KEY")
	if got != "supersecret" {
		t.Errorf("expected %q, got %q", "supersecret", got)
	}
}

// Tests that Load correctly populates the global Config struct C
// using all required and optional environment variables.
func TestLoad_PopulatesConfig(t *testing.T) {
	vars := map[string]string{
		"SECRET_KEY":           "s3cr3t",
		"DB_NAME":              "testdb",
		"DB_USER":              "testuser",
		"DB_PASS":              "testpass",
		"DB_HOST":              "localhost",
		"DB_PORT":              "5433",
		"REDIS_URL":            "redis://localhost:6380",
		"PORT":                 "9000",
		"GOOGLE_CLIENT_ID":     "google-id",
		"GOOGLE_CLIENT_SECRET": "google-secret",
		"GOOGLE_REDIRECT_URL":  "http://localhost/callback",
	}
	for k, v := range vars {
		t.Setenv(k, v)
	}

	Load()

	if C.SecretKey != "s3cr3t" {
		t.Errorf("SecretKey: expected %q, got %q", "s3cr3t", C.SecretKey)
	}
	if C.DBName != "testdb" {
		t.Errorf("DBName: expected %q, got %q", "testdb", C.DBName)
	}
	if C.DBUser != "testuser" {
		t.Errorf("DBUser: expected %q, got %q", "testuser", C.DBUser)
	}
	if C.DBPass != "testpass" {
		t.Errorf("DBPass: expected %q, got %q", "testpass", C.DBPass)
	}
	if C.DBHost != "localhost" {
		t.Errorf("DBHost: expected %q, got %q", "localhost", C.DBHost)
	}
	if C.DBPort != "5433" {
		t.Errorf("DBPort: expected %q, got %q", "5433", C.DBPort)
	}
	if C.RedisURL != "redis://localhost:6380" {
		t.Errorf("RedisURL: expected %q, got %q", "redis://localhost:6380", C.RedisURL)
	}
	if C.Port != "9000" {
		t.Errorf("Port: expected %q, got %q", "9000", C.Port)
	}
	if C.GoogleClientID != "google-id" {
		t.Errorf("GoogleClientID: expected %q, got %q", "google-id", C.GoogleClientID)
	}
	if C.GoogleClientSecret != "google-secret" {
		t.Errorf("GoogleClientSecret: expected %q, got %q", "google-secret", C.GoogleClientSecret)
	}
	if C.GoogleRedirectURL != "http://localhost/callback" {
		t.Errorf("GoogleRedirectURL: expected %q, got %q", "http://localhost/callback", C.GoogleRedirectURL)
	}
}

// Tests that Load falls back to default values for DB_PORT,
// REDIS_URL, and PORT when those env vars are not set.
func TestLoad_DefaultValues(t *testing.T) {
	required := map[string]string{
		"SECRET_KEY":           "s",
		"DB_NAME":              "d",
		"DB_USER":              "u",
		"DB_PASS":              "p",
		"DB_HOST":              "h",
		"GOOGLE_CLIENT_ID":     "gid",
		"GOOGLE_CLIENT_SECRET": "gsec",
		"GOOGLE_REDIRECT_URL":  "gurl",
	}
	optionals := []string{"DB_PORT", "REDIS_URL", "PORT"}
	for _, k := range optionals {
		os.Unsetenv(k)
	}
	for k, v := range required {
		t.Setenv(k, v)
	}

	Load()

	if C.DBPort != "5432" {
		t.Errorf("DBPort default: expected %q, got %q", "5432", C.DBPort)
	}
	if C.RedisURL != "redis://127.0.0.1:6379" {
		t.Errorf("RedisURL default: expected %q, got %q", "redis://127.0.0.1:6379", C.RedisURL)
	}
	if C.Port != "8000" {
		t.Errorf("Port default: expected %q, got %q", "8000", C.Port)
	}
}
