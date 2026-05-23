package utils

import "testing"

func TestPasswordHashAndCheck(t *testing.T) {
	password := "secret123"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}

	if hash == "" {
		t.Fatal("expected hash")
	}

	if err := CheckPassword(hash, password); err != nil {
		t.Fatal("password should match")
	}

	if CheckPassword(hash, "wrong") == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestJWTGenerateAndVerify(t *testing.T) {
	token, err := GenerateToken("test@mail.com", "admin")
	if err != nil {
		t.Fatalf("token error: %v", err)
	}

	claims, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}

	if claims.Email != "test@mail.com" {
		t.Fatal("email mismatch")
	}

	if claims.Role != "admin" {
		t.Fatal("role mismatch")
	}
}