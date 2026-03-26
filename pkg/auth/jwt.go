package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	OpenID string `json:"open_id"`
	jwt.RegisteredClaims
}

type JWTConfig struct {
	SecretKey     string
	TokenDuration time.Duration
}

type JWTManager struct {
	config *JWTConfig
}

func NewJWTManager(config *JWTConfig) *JWTManager {
	return &JWTManager{config: config}
}

func (m *JWTManager) GenerateToken(userID, openID string) (string, error) {
	claims := &Claims{
		UserID: userID,
		OpenID: openID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.TokenDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "smart-park",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.SecretKey))
}

func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.config.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}

func (m *JWTManager) ValidateToken(tokenString string) error {
	_, err := m.ParseToken(tokenString)
	return err
}
