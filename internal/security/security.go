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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
)

// Default salt for deriving fallback key if ENCRYPTION_KEY is not set and file creation fails
const defaultSalt = "ai-daily-brief-secret-db-salt-value"

func getEncryptionKey() []byte {
	// 1. Check environment variable first
	secret := os.Getenv("ENCRYPTION_KEY")
	if secret != "" {
		hash := sha256.Sum256([]byte(secret))
		return hash[:]
	}

	// 2. Check user's home directory key file (~/.ai_daily_brief/key)
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("[Security] Error retrieving user home directory: %v. Using fallback salt.", err)
		hash := sha256.Sum256([]byte(defaultSalt))
		return hash[:]
	}

	aiKeyDir := filepath.Join(homeDir, ".ai_daily_brief")
	keyPath := filepath.Join(aiKeyDir, "key")

	// Check if key file exists
	if _, err := os.Stat(keyPath); os.IsNotExist(err) {
		// Create ~/.ai_daily_brief directory with user-only permissions (0700)
		if err := os.MkdirAll(aiKeyDir, 0700); err != nil {
			log.Printf("[Security] Error creating directory %s: %v. Using fallback salt.", aiKeyDir, err)
			hash := sha256.Sum256([]byte(defaultSalt))
			return hash[:]
		}

		// Generate random 32-byte key
		newKey := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
			log.Printf("[Security] Error generating random key: %v. Using fallback salt.", err)
			hash := sha256.Sum256([]byte(defaultSalt))
			return hash[:]
		}

		// Write key to file with user-only read/write permissions (0600)
		if err := os.WriteFile(keyPath, newKey, 0600); err != nil {
			log.Printf("[Security] Error writing key to %s: %v. Using fallback salt.", keyPath, err)
			hash := sha256.Sum256([]byte(defaultSalt))
			return hash[:]
		}
		log.Printf("[Security] Created new stable symmetric encryption key at %s", keyPath)
	}

	// Read key from file
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		log.Printf("[Security] Error reading key from %s: %v. Using fallback salt.", keyPath, err)
		hash := sha256.Sum256([]byte(defaultSalt))
		return hash[:]
	}

	hash := sha256.Sum256(keyBytes)
	return hash[:]
}

// Encrypt encrypts plainText to a hex-encoded string containing nonce and cipher text
func Encrypt(plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}

	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	cipherText := gcm.Seal(nonce, nonce, []byte(plainText), nil)
	return hex.EncodeToString(cipherText), nil
}

// Decrypt decrypts a hex-encoded cipherText string back to its original plainText
func Decrypt(hexCipherText string) (string, error) {
	if hexCipherText == "" {
		return "", nil
	}

	cipherText, err := hex.DecodeString(hexCipherText)
	if err != nil {
		return "", err
	}

	key := getEncryptionKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(cipherText) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, actualCipherText := cipherText[:nonceSize], cipherText[nonceSize:]
	plainText, err := gcm.Open(nil, nonce, actualCipherText, nil)
	if err != nil {
		return "", err
	}

	return string(plainText), nil
}
