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
