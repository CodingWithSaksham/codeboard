package auth

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"

	"github.com/creative-computing-society/codeboard/internal/config"
	"github.com/creative-computing-society/codeboard/internal/middleware"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

var (
	oauthCfg     *oauth2.Config
	oauthCfgOnce sync.Once
)

const (
	stateCookieName = "oauth_state"
	stateCookieTTL  = 10 * time.Minute
)

// getOAuthCfg lazily initializes and returns the OAuth2 configuration
// for Google authentication.
//
// It uses sync.Once to ensure the configuration is created exactly once
// during the application's lifecycle.
func getOAuthCfg() *oauth2.Config {
	oauthCfgOnce.Do(func() {
		oauthCfg = &oauth2.Config{
			ClientID:     config.C.GoogleClientID,
			ClientSecret: config.C.GoogleClientSecret,
			RedirectURL:  config.C.GoogleRedirectURL,
			Scopes: []string{
				"https://www.googleapis.com/auth/userinfo.email",
				"https://www.googleapis.com/auth/userinfo.profile",
				"openid",
			},
			Endpoint: google.Endpoint,
		}
	})
	return oauthCfg
}

// generateState creates a cryptographically secure random string
// used as the OAuth2 state parameter for CSRF protection.
//
// Returns:
//
//	string: Base64 URL-encoded random state value
//	error:  If secure random generation fails
func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// generateToken creates and signs a JWT using the HS256 signing method.
//
// The token includes the provided userID as a custom claim and
// sets a 30-day expiration time.
//
// Returns:
//
//	string: Signed JWT token
//	error:  If signing fails
func generateToken(userID uint) (string, error) {
	claims := middleware.Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString([]byte(config.C.SecretKey))
}
