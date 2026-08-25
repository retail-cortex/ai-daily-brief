package security

import (
	"os"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	testKey := "my-secret-test-key-12345"
	os.Setenv("ENCRYPTION_KEY", testKey)
	defer os.Unsetenv("ENCRYPTION_KEY")

	plainText := "test-gemini-api-key-value-999"

	encrypted, err := Encrypt(plainText)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == plainText {
		t.Error("Encrypt returned plain text instead of cipher text")
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plainText {
		t.Errorf("Expected decrypted text to match plainText: got %q, want %q", decrypted, plainText)
	}
}

func TestEncryptDecryptEmpty(t *testing.T) {
	encrypted, err := Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt failed for empty string: %v", err)
	}
	if encrypted != "" {
		t.Errorf("Expected empty string for Encrypt(''), got %q", encrypted)
	}

	decrypted, err := Decrypt("")
	if err != nil {
		t.Fatalf("Decrypt failed for empty string: %v", err)
	}
	if decrypted != "" {
		t.Errorf("Expected empty string for Decrypt(''), got %q", decrypted)
	}
}
