package service

import (
	"errors"
	"swift-seat/internal/models"
	token "swift-seat/internal/pkg/Token"
	"swift-seat/internal/repository"

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

func (s *UserService) Register(name, email, password string) error {
	// هش کردن پسورد
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	if err != nil {
		return err
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hashedPassword), // 👈 تغییر نام فیلد به استراکت جدیدت
		Role:         "user",             // خودش دیفالت داره ولی دستی هم بذاری اوکیه
	}

	return s.repo.CreateUser(user)
}

func (s *UserService) Login(email, password string) (string, error) {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return "", errors.New("ایمیل یا کلمه عبور اشتباه است")
	}

	// 👈 مقایسه با فیلد جدید PasswordHash
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return "", errors.New("ایمیل یا کلمه عبور اشتباه است")
	}

	token, err := s.token.GenerateToken(user.ID, user.Role)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *UserService) UpdateUserRole(userID uint, newRole string) error {
	err := s.repo.UpdateUserRole(userID, newRole)
	if err != nil {
		return err
	}
	return nil
}
