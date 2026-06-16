package middleware

import (
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

func (t *AuthMiddleware) AuthRequired(c *fiber.Ctx) error {
    authHeader := c.Get("Authorization")
    if authHeader == "" {
        return c.Status(401).JSON(fiber.Map{"error": "Missing token"})
    }


    tokenString := strings.TrimPrefix(authHeader, "Bearer ")
    

    claims, err := t.authService.VerifyToken(tokenString)
    if err != nil {
        return c.Status(401).JSON(fiber.Map{"error": "Invalid token"})
    }

   
    c.Locals("user_id", claims.UserID)
    c.Locals("role", claims.Role)

    return c.Next()
}