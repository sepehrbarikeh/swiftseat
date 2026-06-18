package service

import (
	"swift-seat/internal/models"
	token "swift-seat/internal/pkg/Token"
	"swift-seat/internal/pkg/apperrors"
	"swift-seat/internal/repository"

	"net/http"

	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo  *repository.PostgresDB
	token *token.Token
}

func NewUserService(repo *repository.PostgresDB, token *token.Token) *UserService {
	return &UserService{
		repo:  repo,
		token: token,
	}
}

func (s *UserService) Register(name, email, password string) *apperrors.AppError {

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to hash password", err)
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword),
		Role:         "user",
	}

	if appErr := s.repo.CreateUser(user); appErr != nil {
		return appErr
	}

	return nil
}

func (s *UserService) Login(email, password string) (string,string, *apperrors.AppError) {
	user, appErr := s.repo.GetUserByEmail(email)
	if appErr != nil {
		return "","", appErr
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "","", apperrors.New(http.StatusUnauthorized, "Invalid credentials", err)
	}

	tok, err := s.token.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "","", apperrors.New(http.StatusInternalServerError, "Failed to generate token", err)
	}

	return tok,user.Role, nil
}

func (s *UserService) UpdateUserRole(userID uint, newRole string) *apperrors.AppError {
	if appErr := s.repo.UpdateUserRole(userID, newRole); appErr != nil {
		return appErr
	}
	return nil
}
