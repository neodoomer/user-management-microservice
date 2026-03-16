package hasher

import (
	"testing"
)

func TestBcryptHasher_HashAndCompare(t *testing.T) {
	h := NewBcryptHasher()

	password := "securePassword123"
	hashed, err := h.Hash(password)
	if err != nil {
		t.Fatalf("Hash() unexpected error: %v", err)
	}

	if hashed == "" {
		t.Fatal("Hash() returned empty string")
	}
	if hashed == password {
		t.Fatal("Hash() returned plaintext password")
	}

	if err := h.Compare(hashed, password); err != nil {
		t.Fatalf("Compare() should succeed for correct password: %v", err)
	}
}

func TestBcryptHasher_CompareMismatch(t *testing.T) {
	h := NewBcryptHasher()

	hashed, err := h.Hash("correctPassword")
	if err != nil {
		t.Fatalf("Hash() unexpected error: %v", err)
	}

	if err := h.Compare(hashed, "wrongPassword"); err == nil {
		t.Fatal("Compare() should fail for wrong password")
	}
}

func TestBcryptHasher_DifferentHashesForSamePassword(t *testing.T) {
	h := NewBcryptHasher()

	hash1, _ := h.Hash("samePassword")
	hash2, _ := h.Hash("samePassword")

	if hash1 == hash2 {
		t.Fatal("Hash() should produce different hashes due to random salt")
	}
}
