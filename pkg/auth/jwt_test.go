package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/xuanyiying/smart-park/pkg/auth"
)

func generateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) ([]byte, error) {
	privBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privBytes,
	}), nil
}

func encodePublicKeyToPEM(publicKey *rsa.PublicKey) ([]byte, error) {
	pubBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}), nil
}

func TestGenerateAndParseToken(t *testing.T) {
	privateKey, publicKey, err := generateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	privatePEM, err := encodePrivateKeyToPEM(privateKey)
	if err != nil {
		t.Fatalf("failed to encode private key: %v", err)
	}

	publicPEM, err := encodePublicKeyToPEM(publicKey)
	if err != nil {
		t.Fatalf("failed to encode public key: %v", err)
	}

	privateFile, err := os.CreateTemp("", "private*.pem")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(privateFile.Name())
	privateFile.Write(privatePEM)
	privateFile.Close()

	publicFile, err := os.CreateTemp("", "public*.pem")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(publicFile.Name())
	publicFile.Write(publicPEM)
	publicFile.Close()

	config := &auth.JWTConfig{
		PrivateKeyPath: privateFile.Name(),
		PublicKeyPath:  publicFile.Name(),
		TokenDuration:  time.Hour,
	}

	manager, err := auth.NewJWTManager(config)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

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
	privateKey, publicKey, err := generateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	privatePEM, _ := encodePrivateKeyToPEM(privateKey)
	publicPEM, _ := encodePublicKeyToPEM(publicKey)

	privateFile, _ := os.CreateTemp("", "private*.pem")
	defer os.Remove(privateFile.Name())
	privateFile.Write(privatePEM)
	privateFile.Close()

	publicFile, _ := os.CreateTemp("", "public*.pem")
	defer os.Remove(publicFile.Name())
	publicFile.Write(publicPEM)
	publicFile.Close()

	config := &auth.JWTConfig{
		PrivateKeyPath: privateFile.Name(),
		PublicKeyPath:  publicFile.Name(),
		TokenDuration:  -time.Hour,
	}

	manager, err := auth.NewJWTManager(config)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

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
	privateKey, publicKey, err := generateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("failed to generate key pair: %v", err)
	}

	privatePEM, _ := encodePrivateKeyToPEM(privateKey)
	publicPEM, _ := encodePublicKeyToPEM(publicKey)

	privateFile, _ := os.CreateTemp("", "private*.pem")
	defer os.Remove(privateFile.Name())
	privateFile.Write(privatePEM)
	privateFile.Close()

	publicFile, _ := os.CreateTemp("", "public*.pem")
	defer os.Remove(publicFile.Name())
	publicFile.Write(publicPEM)
	publicFile.Close()

	config := &auth.JWTConfig{
		PrivateKeyPath: privateFile.Name(),
		PublicKeyPath:  publicFile.Name(),
		TokenDuration:  time.Hour,
	}

	manager, err := auth.NewJWTManager(config)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}

	_, err = manager.ParseToken("invalid-token-string")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestTokenWithDifferentKeys(t *testing.T) {
	privateKey1, publicKey1, _ := generateRSAKeyPair(2048)
	privateKey2, publicKey2, _ := generateRSAKeyPair(2048)

	privatePEM1, _ := encodePrivateKeyToPEM(privateKey1)
	publicPEM1, _ := encodePublicKeyToPEM(publicKey1)
	privatePEM2, _ := encodePrivateKeyToPEM(privateKey2)
	publicPEM2, _ := encodePublicKeyToPEM(publicKey2)

	privateFile1, _ := os.CreateTemp("", "private1*.pem")
	defer os.Remove(privateFile1.Name())
	privateFile1.Write(privatePEM1)
	privateFile1.Close()

	publicFile1, _ := os.CreateTemp("", "public1*.pem")
	defer os.Remove(publicFile1.Name())
	publicFile1.Write(publicPEM1)
	publicFile1.Close()

	privateFile2, _ := os.CreateTemp("", "private2*.pem")
	defer os.Remove(privateFile2.Name())
	privateFile2.Write(privatePEM2)
	privateFile2.Close()

	publicFile2, _ := os.CreateTemp("", "public2*.pem")
	defer os.Remove(publicFile2.Name())
	publicFile2.Write(publicPEM2)
	publicFile2.Close()

	config1 := &auth.JWTConfig{
		PrivateKeyPath: privateFile1.Name(),
		PublicKeyPath:  publicFile1.Name(),
		TokenDuration:  time.Hour,
	}

	config2 := &auth.JWTConfig{
		PrivateKeyPath: privateFile2.Name(),
		PublicKeyPath:  publicFile2.Name(),
		TokenDuration:  time.Hour,
	}

	manager1, err := auth.NewJWTManager(config1)
	if err != nil {
		t.Fatalf("failed to create manager1: %v", err)
	}

	manager2, err := auth.NewJWTManager(config2)
	if err != nil {
		t.Fatalf("failed to create manager2: %v", err)
	}

	token, err := manager1.GenerateToken("user123", "open123")
	if err != nil {
		t.Fatalf("failed to generate token: %v", err)
	}

	_, err = manager2.ParseToken(token)
	if err == nil {
		t.Error("expected error when parsing token with different key")
	}
}
