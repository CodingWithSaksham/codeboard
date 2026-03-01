package middleware

import (
	"github.com/creative-computing-society/codeboard/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

// Claims is the JWT payload.
type Claims struct {
	UserID uint `json:"user_id"`
	jwt.RegisteredClaims
}

// RequireAuth validates the Bearer JWT and stores only the userID in Locals.
//
// Previously this loaded the full CUser from Postgres on every request.
// The JWT already contains the user ID — that's the only thing any downstream
// handler actually needs before it runs its own targeted query. Removing the
// DB hit here cuts one round-trip from every authenticated endpoint.
//
// Access the ID with: c.Locals("userID").(uint)
func RequireAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if len(authHeader) < 8 {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Authorization header missing"})
	}
	// Avoid SplitN allocation: the prefix is always exactly "Bearer " (7 bytes).
	if authHeader[:7] != "Bearer " {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid authorization format"})
	}
	tokenStr := authHeader[7:]

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fiber.ErrUnauthorized
		}
		return []byte(config.C.SecretKey), nil
	})
	if err != nil || !token.Valid {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
	}

	c.Locals("userID", claims.UserID)
	return c.Next()
}
