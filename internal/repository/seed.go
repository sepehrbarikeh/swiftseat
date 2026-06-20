package repository

import (
	"log"
	"golang.org/x/crypto/bcrypt"
)

func (p *PostgresDB) SeedAdmin() {
	var count int64
	p.DB.Table("users").Where("role = ?", "admin").Count(&count)

	if count > 0 {
		log.Println("Admin already exists, skipping...")
		return
	}

	
	password := "adminadmin"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password")
	}


	admin := map[string]interface{}{
		"name":          "Admin",
		"email":         "admin@swiftseat.com",
		"password_hash":      string(hashedPassword),
		"role":          "admin",
	}

	if err := p.DB.Table("users").Create(&admin).Error; err != nil {
		log.Fatal("Failed to seed admin:", err)
	}

	log.Println("✅ Admin seeded successfully!")
}