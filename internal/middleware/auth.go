package middleware

import "github.com/gofiber/fiber/v2"

func (t *AuthMiddleware) AdminOnly(c *fiber.Ctx) error {
	// ۱. دریافت مقدار از Locals
	val := c.Locals("role")

	// ۲. بررسی وجود مقدار (Nil check)
	if val == nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Authentication required",
		})
	}

	// ۳. بررسی نوع (Type assertion safe)
	userRole, ok := val.(string)
	if !ok || userRole != "admin" {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "Access denied: Admins only",
		})
	}

	return c.Next()
}
