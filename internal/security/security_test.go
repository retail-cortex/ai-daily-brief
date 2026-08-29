// Copyright 2026 Retail Cortex
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package security

import (
	"context"
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

func TestEncryptDecryptWithContext(t *testing.T) {
	testKey := "context-key-54321"
	os.Setenv("ENCRYPTION_KEY", testKey)
	defer os.Unsetenv("ENCRYPTION_KEY")

	ctx := context.Background()
	plainText := "vertex-project-credential-token"

	encrypted, err := EncryptWithContext(ctx, plainText)
	if err != nil {
		t.Fatalf("EncryptWithContext failed: %v", err)
	}

	decrypted, err := DecryptWithContext(ctx, encrypted)
	if err != nil {
		t.Fatalf("DecryptWithContext failed: %v", err)
	}

	if decrypted != plainText {
		t.Errorf("Decrypted %q != expected %q", decrypted, plainText)
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

func TestGetSecretFallbackEnv(t *testing.T) {
	os.Setenv("GEMINI_API_KEY", "ai-studio-sample-api-key-test")
	defer os.Unsetenv("GEMINI_API_KEY")

	ctx := context.Background()
	secretVal, err := GetSecret(ctx, "gemini-api-key")
	if err != nil {
		t.Fatalf("GetSecret fallback failed: %v", err)
	}

	if secretVal != "ai-studio-sample-api-key-test" {
		t.Errorf("GetSecret = %q; want %q", secretVal, "ai-studio-sample-api-key-test")
	}
}

func TestGetGCPProjectID(t *testing.T) {
	orig := os.Getenv("GOOGLE_CLOUD_PROJECT")
	defer func() {
		if orig != "" {
			os.Setenv("GOOGLE_CLOUD_PROJECT", orig)
		} else {
			os.Unsetenv("GOOGLE_CLOUD_PROJECT")
		}
		ResetProjectIDCache()
	}()

	ResetProjectIDCache()
	os.Setenv("GOOGLE_CLOUD_PROJECT", "custom-gcp-project-123")
	if proj := GetGCPProjectID(); proj != "custom-gcp-project-123" {
		t.Errorf("GetGCPProjectID() = %q; want %q", proj, "custom-gcp-project-123")
	}
}
