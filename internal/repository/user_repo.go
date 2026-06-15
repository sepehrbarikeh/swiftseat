package repository

import (
	"errors"
	"swift-seat/internal/models"
)

func (p *PostgresDB) CreateUser(user *models.User) error {
	return p.DB.Create(user).Error
}

// GetUserByEmail یک کاربر را بر اساس ایمیل پیدا می‌کند
func (p *PostgresDB) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := p.DB.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (p *PostgresDB) UpdateUserRole(userID uint, newRole string) error {
	// پیدا کردن یوزر و آپدیت کردن فیلد Role
	result := p.DB.Model(&models.User{}).Where("id = ?", userID).Update("role", newRole)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("user not found")
	}
	return nil
}
