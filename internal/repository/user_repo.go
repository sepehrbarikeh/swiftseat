package repository

import (
	"errors"
	"net/http"
	"swift-seat/internal/models"
	"swift-seat/internal/pkg/apperrors"

	"gorm.io/gorm"
)

func (p *PostgresDB) CreateUser(user *models.User) *apperrors.AppError {
	if err := p.DB.Create(user).Error; err != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to create user", err)
	}
	return nil
}

// GetUserByEmail یک کاربر را بر اساس ایمیل پیدا می‌کند
func (p *PostgresDB) GetUserByEmail(email string) (*models.User, *apperrors.AppError) {
	var user models.User
	err := p.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperrors.New(http.StatusNotFound, "User not found", err)
		}
		return nil, apperrors.New(http.StatusInternalServerError, "DB error", err)
	}
	return &user, nil
}

func (p *PostgresDB) UpdateUserRole(userID uint, newRole string) *apperrors.AppError {
	// پیدا کردن یوزر و آپدیت کردن فیلد Role
	result := p.DB.Model(&models.User{}).Where("id = ?", userID).Update("role", newRole)
	if result.Error != nil {
		return apperrors.New(http.StatusInternalServerError, "Failed to update user role", result.Error)
	}
	if result.RowsAffected == 0 {
		return apperrors.New(http.StatusNotFound, "User not found", errors.New("user not found"))
	}
	return nil
}
