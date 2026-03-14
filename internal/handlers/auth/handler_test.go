package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/creative-computing-society/codeboard/internal/config"
	"github.com/gofiber/fiber/v2"
)

// newTestApp creates a minimal Fiber app wired with the auth routes
// for use across handler tests.
func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			return c.Status(fiber.StatusInternalServerError).
				JSON(fiber.Map{"error": err.Error()})
		},
	})
	app.Get("/auth/google/login", GoogleLogin)
	app.Get("/auth/google/callback", GoogleCallback)
	return app
}

// setTestOAuthConfig sets dummy OAuth config values for tests that
// need getOAuthCfg() to return a non-nil object.
func setTestOAuthConfig() {
	config.C.GoogleClientID = "test-id"
	config.C.GoogleClientSecret = "test-secret"
	config.C.GoogleRedirectURL = "http://localhost/callback"
}

// assertJSONError decodes the response body and checks that the
// "error" field matches want exactly.
func assertJSONError(t *testing.T, body io.Reader, want string) {
	t.Helper()
	var m map[string]string
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		t.Fatalf("failed to decode JSON body: %v", err)
	}
	if got := m["error"]; got != want {
		t.Errorf("error field: expected %q, got %q", want, got)
	}
}

// Tests that GET /auth/google/login responds with a 307 redirect
// and sets the oauth_state cookie.
func TestGoogleLogin_RedirectsWithStateCookie(t *testing.T) {
	oauthCfgOnce = sync.Once{}
	oauthCfg = nil
	setTestOAuthConfig()

	app := newTestApp()
	req := httptest.NewRequest("GET", "/auth/google/login", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusTemporaryRedirect {
		t.Errorf("expected 307, got %d", resp.StatusCode)
	}

	var found bool
	for _, c := range resp.Cookies() {
		if c.Name == stateCookieName {
			found = true
			if c.Value == "" {
				t.Error("oauth_state cookie is empty")
			}
			if !c.HttpOnly {
				t.Error("oauth_state cookie should be HttpOnly")
			}
		}
	}
	if !found {
		t.Error("oauth_state cookie not set in response")
	}
}

// Tests that GET /auth/google/callback without a state cookie
// returns 401 Unauthorized.
func TestGoogleCallback_MissingStateCookie_Returns401(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest("GET", "/auth/google/callback?state=somestate&code=somecode", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	assertJSONError(t, resp.Body, "Invalid OAuth state")
}

// Tests that GET /auth/google/callback with a mismatched state
// parameter returns 401 with an "Invalid OAuth state" error.
func TestGoogleCallback_StateMismatch_Returns401(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest("GET", "/auth/google/callback?state=wrong&code=abc", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "correct"})

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("expected 401, got %d", resp.StatusCode)
	}
	assertJSONError(t, resp.Body, "Invalid OAuth state")
}

// Tests that GET /auth/google/callback with matching state but
// missing code query param returns 400 Bad Request.
func TestGoogleCallback_MissingCode_Returns400(t *testing.T) {
	app := newTestApp()
	req := httptest.NewRequest("GET", "/auth/google/callback?state=mystate", nil)
	req.AddCookie(&http.Cookie{Name: stateCookieName, Value: "mystate"})

	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusBadRequest {
		t.Errorf("expected 400, got %d", resp.StatusCode)
	}
	assertJSONError(t, resp.Body, "Missing code parameter")
}
