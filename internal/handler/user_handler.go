package handlers

import (
	"net/http"
	"strconv"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/pkg/utils"
	"swift-seat/internal/service"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	svc *service.UserService
}

type RegisterDTO struct {
	Name     string `json:"name" validate:"required,min=3"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginDTO struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
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

	if err := c.BodyParser(&req); err != nil {
		appErr := apperrors.NewValidationError("Invalid request body")
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	if errs := utils.ValidateStruct(req); errs != nil {
		return c.Status(422).JSON(errs)
	}

	if appErr := h.svc.Register(req.Name, req.Email, req.Password); appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
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
		appErr := apperrors.NewValidationError("Invalid request body")
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	if errs := utils.ValidateStruct(req); errs != nil {
        return c.Status(422).JSON(errs)
    }

	token,role, appErr := h.svc.Login(req.Email, req.Password)
	if appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"status": "success",
		"token":  token,
		"type":   "Bearer",
		"role" : role,
	})
}

// @Summary Change user role
// @Description Update the role of a user account
// @Tags Users
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param role body UpdateRoleRequest true "Role change request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/users/{id}/role [post]
func (h *UserHandler) ChangeUserRole(c *fiber.Ctx) error {
	userID := c.Params("id")
	idUint, _ := strconv.Atoi(userID)

	var req UpdateRoleRequest
	if err := c.BodyParser(&req); err != nil {
		appErr := apperrors.NewValidationError("Invalid request")
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	if errs := utils.ValidateStruct(req); errs != nil {
        return c.Status(422).JSON(errs)
    }

	if appErr := h.svc.UpdateUserRole(uint(idUint), req.Role); appErr != nil {
		return c.Status(appErr.StatusCode).JSON(appErr)
	}

	return c.Status(200).JSON(fiber.Map{"message": "Role updated successfully"})
}
