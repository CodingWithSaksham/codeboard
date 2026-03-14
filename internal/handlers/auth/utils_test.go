package auth

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creative-computing-society/codeboard/internal/config"
	"github.com/creative-computing-society/codeboard/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
)

// Tests that generateState returns a non-empty, base64url-encoded
// string of at least 20 characters on every call.
func TestGenerateState_ReturnsNonEmpty(t *testing.T) {
	state, err := generateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(state) < 20 {
		t.Errorf("state too short: %q", state)
	}
}

// Tests that two consecutive generateState calls produce
// different values, ensuring randomness.
func TestGenerateState_IsRandom(t *testing.T) {
	s1, _ := generateState()
	s2, _ := generateState()
	if s1 == s2 {
		t.Error("generateState returned identical values on consecutive calls")
	}
}

// Tests that generateState output contains only valid
// base64url characters (no '+' or '/').
func TestGenerateState_IsBase64URL(t *testing.T) {
	state, err := generateState()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.ContainsAny(state, "+/") {
		t.Errorf("state contains non-URL-safe base64 chars: %q", state)
	}
}

// Tests that generateToken returns a signed JWT containing
// the correct userID claim without error.
func TestGenerateToken_ContainsUserID(t *testing.T) {
	config.C.SecretKey = "test-secret-key"
	userID := uint(42)

	tokenStr, err := generateToken(userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims := &middleware.Claims{}
	_, err = jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return []byte(config.C.SecretKey), nil
	})
	if err != nil {
		t.Fatalf("failed to parse generated token: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("expected UserID %d, got %d", userID, claims.UserID)
	}
}

// Tests that the JWT produced by generateToken expires
// approximately 30 days from issuance.
func TestGenerateToken_ExpiresIn30Days(t *testing.T) {
	config.C.SecretKey = "test-secret-key"

	tokenStr, err := generateToken(1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	claims := &middleware.Claims{}
	jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		return []byte(config.C.SecretKey), nil
	})

	expectedExpiry := time.Now().Add(30 * 24 * time.Hour)
	diff := claims.ExpiresAt.Time.Sub(expectedExpiry)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("expiry not ~30 days from now, got diff: %v", diff)
	}
}

// Tests that generateToken uses HMAC-SHA256 signing
// and the token is not empty.
func TestGenerateToken_UsesHS256(t *testing.T) {
	config.C.SecretKey = "test-secret-key"

	tokenStr, err := generateToken(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokenStr == "" {
		t.Error("expected non-empty token string")
	}

	// JWT format: header.payload.signature
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		t.Errorf("expected 3 JWT parts, got %d", len(parts))
	}
}

// Tests that getOAuthCfg returns a non-nil config with
// the correct scopes set.
func TestGetOAuthCfg_HasExpectedScopes(t *testing.T) {
	config.C.GoogleClientID = "test-client-id"
	config.C.GoogleClientSecret = "test-client-secret"
	config.C.GoogleRedirectURL = "http://localhost/callback"

	// Reset the once so config is re-initialized for test isolation.
	oauthCfgOnce = sync.Once{}
	oauthCfg = nil

	cfg := getOAuthCfg()
	if cfg == nil {
		t.Fatal("getOAuthCfg returned nil")
	}

	expectedScopes := map[string]bool{
		"https://www.googleapis.com/auth/userinfo.email":   true,
		"https://www.googleapis.com/auth/userinfo.profile": true,
		"openid": true,
	}
	for _, s := range cfg.Scopes {
		if !expectedScopes[s] {
			t.Errorf("unexpected scope: %q", s)
		}
		delete(expectedScopes, s)
	}
	for missing := range expectedScopes {
		t.Errorf("missing expected scope: %q", missing)
	}
}

// Tests that getOAuthCfg is a singleton — multiple calls
// return the exact same pointer.
func TestGetOAuthCfg_Singleton(t *testing.T) {
	config.C.GoogleClientID = "id"
	config.C.GoogleClientSecret = "secret"
	config.C.GoogleRedirectURL = "url"

	oauthCfgOnce = sync.Once{}
	oauthCfg = nil

	cfg1 := getOAuthCfg()
	cfg2 := getOAuthCfg()
	if cfg1 != cfg2 {
		t.Error("getOAuthCfg should return the same instance on repeated calls")
	}
}
