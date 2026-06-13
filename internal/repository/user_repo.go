package repository

import "swift-seat/internal/models"



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
