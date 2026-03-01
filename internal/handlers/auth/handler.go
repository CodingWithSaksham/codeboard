package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/creative-computing-society/codeboard/internal/db"
	"github.com/creative-computing-society/codeboard/internal/models"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/oauth2"
	googleoauth "google.golang.org/api/oauth2/v2"
	"google.golang.org/api/option"
)

// GoogleLogin initiates the OAuth2 authentication flow with Google.
//
// It generates a CSRF protection state value, stores it in a secure
// HTTP-only cookie with a limited lifetime, and redirects the client
// to Google's OAuth2 authorization URL.
//
// Returns:
//
//	500: State generation failed
//
// Route:
//
//	GET /auth/google/login
func GoogleLogin(c *fiber.Ctx) error {
	state, err := generateState()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to generate state"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     stateCookieName,
		Value:    state,
		Expires:  time.Now().Add(stateCookieTTL),
		HTTPOnly: true,
		SameSite: "Lax",
	})

	return c.Redirect(
		getOAuthCfg().AuthCodeURL(state, oauth2.AccessTypeOnline),
		fiber.StatusTemporaryRedirect,
	)
}

// GoogleCallback processes the OAuth2 callback from Google.
//
// It validates the state parameter, exchanges the authorization
// code for tokens, fetches user information, and creates or
// retrieves the corresponding user.
//
// Returns:
//
//	200: JWT issued successfully
//	400: Code parameter is missing
//	401: Invalid state, code exchange failed, or email not verified
//	500: OAuth service creation failed, user info fetch failed, or user upsert failed
//
// Route:
//
//	GET /auth/google/callback
func GoogleCallback(c *fiber.Ctx) error {
	// Validate CSRF state
	cookieState := c.Cookies(stateCookieName)
	if cookieState == "" || cookieState != c.Query("state") {
		return c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "Invalid OAuth state"})
	}

	// Reset state cookie
	c.Cookie(&fiber.Cookie{
		Name:    stateCookieName,
		Value:   "",
		Expires: time.Now().Add(-time.Hour),
	})

	// Exchange auth code for token
	code := c.Query("code")
	if code == "" {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"error": "Missing code parameter"})
	}

	ctx := context.Background()
	token, err := getOAuthCfg().Exchange(ctx, code)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": fmt.Sprintf("Code exchange failed: %v", err)})
	}

	// Fetch user info via Google API client
	oauthClient := getOAuthCfg().Client(ctx, token)
	svc, err := googleoauth.NewService(ctx, option.WithHTTPClient(oauthClient))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to create OAuth service"})
	}

	userInfo, err := svc.Userinfo.Get().Do()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": fmt.Sprintf("Failed to fetch user info: %v", err)})
	}

	if !*userInfo.VerifiedEmail {
		return c.Status(fiber.StatusUnauthorized).
			JSON(fiber.Map{"error": "Google email not verified"})
	}

	// Upsert user
	user := models.CUser{
		GoogleID:  userInfo.Id,
		Email:     userInfo.Email,
		Username:  userInfo.Name,
		AvatarURL: userInfo.Picture,
	}

	result := db.DB.
		Where(models.CUser{GoogleID: userInfo.Id}).
		Assign(models.CUser{
			Email:     userInfo.Email,
			Username:  userInfo.Name,
			AvatarURL: userInfo.Picture,
		}).
		FirstOrCreate(&user)

	if result.Error != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to upsert user"})
	}

	// Issue JWT
	jwtToken, err := generateToken(user.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).
			JSON(fiber.Map{"error": "Failed to generate token"})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"token": jwtToken})
}
