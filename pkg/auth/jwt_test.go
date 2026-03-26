package auth_test

import (
	"testing"
	"time"

	"github.com/xuanyiying/smart-park/pkg/auth"
)

func TestGenerateAndParseToken(t *testing.T) {
	config := &auth.JWTConfig{
		SecretKey:     "test-secret-key-with-at-least-32-characters",
		TokenDuration: time.Hour,
	}

	manager := auth.NewJWTManager(config)

	token, err := manager.GenerateToken("user123", "open123")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	if token == "" {
		t.Error("token should not be empty")
	}

	claims, err := manager.ParseToken(token)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}

	if claims.UserID != "user123" {
		t.Errorf("expected user_id user123, got %s", claims.UserID)
	}

	if claims.OpenID != "open123" {
		t.Errorf("expected open_id open123, got %s", claims.OpenID)
	}
}

func TestExpiredToken(t *testing.T) {
	config := &auth.JWTConfig{
		SecretKey:     "test-secret-key-with-at-least-32-characters",
		TokenDuration: -time.Hour,
	}

	manager := auth.NewJWTManager(config)

	token, err := manager.GenerateToken("user123", "open123")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = manager.ParseToken(token)
	if err == nil {
		t.Error("expected error for expired token")
	}
}

func TestInvalidToken(t *testing.T) {
	config := &auth.JWTConfig{
		SecretKey:     "test-secret-key-with-at-least-32-characters",
		TokenDuration: time.Hour,
	}

	manager := auth.NewJWTManager(config)

	_, err := manager.ParseToken("invalid-token-string")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestTokenWithDifferentSecret(t *testing.T) {
	config1 := &auth.JWTConfig{
		SecretKey:     "secret-key-1-with-at-least-32-characters",
		TokenDuration: time.Hour,
	}

	config2 := &auth.JWTConfig{
		SecretKey:     "secret-key-2-with-at-least-32-characters",
		TokenDuration: time.Hour,
	}

	manager1 := auth.NewJWTManager(config1)
	manager2 := auth.NewJWTManager(config2)

	token, err := manager1.GenerateToken("user123", "open123")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = manager2.ParseToken(token)
	if err == nil {
		t.Error("expected error when parsing token with different secret")
	}
}
