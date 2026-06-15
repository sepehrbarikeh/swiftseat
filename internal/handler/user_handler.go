package handlers

import (
	"net/http"
	"strconv"
	"swift-seat/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	svc *service.UserService
}

type RegisterDTO struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginDTO struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateRoleRequest struct {
    Role string `json:"role" validate:"required,oneof=admin user"`
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	return &UserHandler{svc: svc}
}

// @Summary Register a new user
// @Description Register a new user account
// @Tags Authentication
// @Accept json
// @Produce json
// @Param user body RegisterDTO true "User registration"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Router /api/register [post]
func (h *UserHandler) Register(c *fiber.Ctx) error {
	var req RegisterDTO

	// پارس کردن جی‌سان ورودی
	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	// ولیدیشن ساده الزامی بودن فیلدها
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "All fields (name, email, and password) are required."})
	}

	// صدا زدن لایه سرویس
	if err := h.svc.Register(req.Name, req.Email, req.Password); err != nil {
		// معمولاً ارور دیتابیس به خاطر ایمیل تکراری رخ میده
		return c.Status(http.StatusConflict).JSON(fiber.Map{"error": "Registration failed. This email address is probably already registered."})
	}

	return c.Status(http.StatusCreated).JSON(fiber.Map{
		"status":  "success",
		"message": "Your account has been successfully created.",
	})
}

// @Summary Login user
// @Description Authenticate a user and return a JWT token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param credentials body LoginDTO true "Login request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/login [post]
func (h *UserHandler) Login(c *fiber.Ctx) error {
	var req LoginDTO

	if err := c.BodyParser(&req); err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "Bad Request"})
	}

	if req.Email == "" || req.Password == "" {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "email and password is required."})
	}

	// صدا زدن لایه سرویس برای تایید هویت و گرفتن توکن JWT
	token, err := h.svc.Login(req.Email, req.Password)
	if err != nil {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{"error": err.Error()})
	}

	// برگرداندن توکن به همراه نوع آن برای فرانت‌آند
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "success",
		"token":  token,
		"type":   "Bearer",
	})
}


func (h *UserHandler) ChangeUserRole(c *fiber.Ctx) error {
    userID := c.Params("id")
    idUint, _ := strconv.Atoi(userID)

    var req UpdateRoleRequest
    if err := c.BodyParser(&req); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
    }

    // ولیدیشن ساده (اگر از پکیج validator استفاده می‌کنی اینجا فراخوانی کن)
    if req.Role != "admin" && req.Role != "user" {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid role"})
    }

    if err := h.svc.UpdateUserRole(uint(idUint), req.Role); err != nil {
        return c.Status(500).JSON(fiber.Map{"error": err.Error()})
    }

    return c.Status(200).JSON(fiber.Map{"message": "Role updated successfully"})
}