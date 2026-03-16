package token

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	mgr := NewJWTManager("test-secret-key", 15*time.Minute)

	userID := uuid.New()
	role := "admin"

	tokenStr, err := mgr.Generate(userID, role)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("Generate() returned empty token")
	}

	claims, err := mgr.Validate(tokenStr)
	if err != nil {
		t.Fatalf("Validate() error: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.Role != role {
		t.Errorf("Role = %v, want %v", claims.Role, role)
	}
}

func TestJWTManager_ValidateExpired(t *testing.T) {
	mgr := NewJWTManager("test-secret-key", -1*time.Second)

	tokenStr, err := mgr.Generate(uuid.New(), "user")
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	_, err = mgr.Validate(tokenStr)
	if err == nil {
		t.Fatal("Validate() should fail for expired token")
	}
}

func TestJWTManager_ValidateWrongSecret(t *testing.T) {
	mgr1 := NewJWTManager("secret-one", 15*time.Minute)
	mgr2 := NewJWTManager("secret-two", 15*time.Minute)

	tokenStr, _ := mgr1.Generate(uuid.New(), "user")

	_, err := mgr2.Validate(tokenStr)
	if err == nil {
		t.Fatal("Validate() should fail with wrong secret")
	}
}

func TestJWTManager_ValidateGarbage(t *testing.T) {
	mgr := NewJWTManager("test-secret", 15*time.Minute)

	_, err := mgr.Validate("not.a.valid.token")
	if err == nil {
		t.Fatal("Validate() should fail for garbage input")
	}
}
