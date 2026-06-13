package middleware

import (
	"net/http"
	"strings"
	"swift-seat/internal/pkg/Token"

	"github.com/gofiber/fiber/v2"
)

type AuthMiddleware struct {
	authService *token.Token
}

func NewAuthMiddleware(authService *token.Token) *AuthMiddleware {
	return &AuthMiddleware{
		authService: authService,
	}
}

func (t *AuthMiddleware) AuthRequired() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized:"})
		}

		// ۲. جداسازی کلمه Bearer از اصل توکن
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid token format"})
		}

		tokenString := parts[1]

		claims, err := t.authService.VerifyToken(tokenString)
		if err != nil {
			return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": "Invalid or expired token"})
		}

		
		c.Locals("userID", claims.UserID)

		return c.Next() 
	}
}