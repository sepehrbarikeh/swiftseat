package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Token struct{
	secretKey string
}

func New(secretKey string) *Token {
	return &Token{
		secretKey: secretKey,
	}
}


type JWTClaims struct {
	UserID uint `json:"user_id"`
	Role string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken یک توکن جدید با عمر ۷ روز برای کاربر می‌سازد
func (t Token) GenerateToken(userID uint,role string) (string, error) {
	claims := JWTClaims{
		UserID: userID,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(t.secretKey))
}

// VerifyToken توکن دریافتی را بررسی و کلمز (دیتا) داخلش را برمی‌گرداند
func (t Token) VerifyToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(t.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}