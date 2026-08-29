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
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

// Default salt for deriving fallback key if no key is found in GCP Secret Manager, env, or disk
const defaultSalt = "ai-daily-brief-secret-db-salt-value"
const defaultKeySecretName = "ai-daily-brief-encryption-key"

var (
	smClient     *secretmanager.Client
	smClientOnce sync.Once
	smClientErr  error

	cachedProjectID string
	projectIDOnce   sync.Once
)

// getSecretManagerClient returns a singleton Secret Manager client instance
func getSecretManagerClient(ctx context.Context) (*secretmanager.Client, error) {
	smClientOnce.Do(func() {
		// Use a dedicated background context with timeout for client initialization
		initCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		smClient, smClientErr = secretmanager.NewClient(initCtx)
	})
	return smClient, smClientErr
}

// GetGCPProjectID resolves the Google Cloud Project ID dynamically with fallback priority:
// 1. Environment variables: GOOGLE_CLOUD_PROJECT, GCP_PROJECT, GCLOUD_PROJECT
// 2. GCP Compute Metadata Server (when running on Cloud Run, GKE, GCE)
// 3. Local gcloud CLI active configuration (`gcloud config get-value project`)
func GetGCPProjectID() string {
	projectIDOnce.Do(func() {
		// 1. Environment variables
		if proj := strings.TrimSpace(os.Getenv("GOOGLE_CLOUD_PROJECT")); proj != "" {
			cachedProjectID = proj
			return
		}
		if proj := strings.TrimSpace(os.Getenv("GCP_PROJECT")); proj != "" {
			cachedProjectID = proj
			return
		}
		if proj := strings.TrimSpace(os.Getenv("GCLOUD_PROJECT")); proj != "" {
			cachedProjectID = proj
			return
		}

		// 2. GCP Metadata Server (Cloud Run / GKE / Compute Engine)
		if metadata.OnGCE() {
			if proj, err := metadata.ProjectID(); err == nil && strings.TrimSpace(proj) != "" {
				cachedProjectID = strings.TrimSpace(proj)
				return
			}
		}

		// 3. Local gcloud CLI configuration
		cmd := exec.Command("gcloud", "config", "get-value", "project")
		if out, err := cmd.Output(); err == nil {
			proj := strings.TrimSpace(string(out))
			if proj != "" && !strings.Contains(proj, " ") && proj != "(unset)" {
				cachedProjectID = proj
				return
			}
		}
	})
	return cachedProjectID
}

// ResetProjectIDCache resets the cached project ID (useful for testing)
func ResetProjectIDCache() {
	cachedProjectID = ""
	projectIDOnce = sync.Once{}
}

// GetSecret fetches a secret payload string from Google Cloud Secret Manager.
// secretNameOrID can be either:
// 1. A full resource name: "projects/{project}/secrets/{secret}/versions/{version}"
// 2. A simple secret ID: "my-secret" (which resolves against GetGCPProjectID() and accesses "latest")
// If GCP Secret Manager is unreachable or not configured, it gracefully falls back to environment variables.
func GetSecret(ctx context.Context, secretNameOrID string) (string, error) {
	if strings.TrimSpace(secretNameOrID) == "" {
		return "", errors.New("secret name cannot be empty")
	}

	// 1. Try Google Cloud Secret Manager if available
	client, err := getSecretManagerClient(ctx)
	if err == nil && client != nil {
		var resourceName string
		if strings.HasPrefix(secretNameOrID, "projects/") {
			resourceName = secretNameOrID
		} else {
			projectID := GetGCPProjectID()
			if projectID != "" {
				resourceName = fmt.Sprintf("projects/%s/secrets/%s/versions/latest", projectID, secretNameOrID)
			}
		}

		if resourceName != "" {
			reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			req := &secretmanagerpb.AccessSecretVersionRequest{
				Name: resourceName,
			}
			resp, accessErr := client.AccessSecretVersion(reqCtx, req)
			if accessErr == nil && resp != nil && resp.Payload != nil {
				return string(resp.Payload.Data), nil
			}
			log.Printf("[Security] GCP Secret Manager access notice (%s): %v. Attempting local env fallback.", secretNameOrID, accessErr)
		}
	}

	// 2. Fallback to environment variables
	if val := os.Getenv(secretNameOrID); val != "" {
		return val, nil
	}
	normalizedEnv := strings.ToUpper(strings.ReplaceAll(secretNameOrID, "-", "_"))
	if val := os.Getenv(normalizedEnv); val != "" {
		return val, nil
	}

	return "", fmt.Errorf("secret %q not found in Secret Manager or environment variables", secretNameOrID)
}

// SetSecret creates or updates a secret and adds a new payload version in Google Cloud Secret Manager
func SetSecret(ctx context.Context, projectID, secretID string, payload []byte) error {
	if projectID == "" {
		projectID = GetGCPProjectID()
	}
	if projectID == "" {
		return errors.New("GCP Project ID is required to set secrets in Secret Manager")
	}
	if secretID == "" {
		return errors.New("secret ID cannot be empty")
	}

	client, err := getSecretManagerClient(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize Secret Manager client: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	parent := fmt.Sprintf("projects/%s", projectID)
	secretName := fmt.Sprintf("%s/secrets/%s", parent, secretID)

	// Check if secret exists; if not, create it
	_, err = client.GetSecret(reqCtx, &secretmanagerpb.GetSecretRequest{
		Name: secretName,
	})
	if err != nil {
		_, err = client.CreateSecret(reqCtx, &secretmanagerpb.CreateSecretRequest{
			Parent:   parent,
			SecretId: secretID,
			Secret: &secretmanagerpb.Secret{
				Replication: &secretmanagerpb.Replication{
					Replication: &secretmanagerpb.Replication_Automatic_{
						Automatic: &secretmanagerpb.Replication_Automatic{},
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create secret %s: %w", secretID, err)
		}
	}

	// Add new secret version
	_, err = client.AddSecretVersion(reqCtx, &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretName,
		Payload: &secretmanagerpb.SecretPayload{
			Data: payload,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to add secret version to %s: %w", secretID, err)
	}

	return nil
}

// getEncryptionKey resolves the symmetric AES-256 encryption key with priority:
// 1. Google Cloud Secret Manager secret (`ai-daily-brief-encryption-key` or `ENCRYPTION_KEY_SECRET_NAME`)
// 2. Environment variable `ENCRYPTION_KEY`
// 3. Local home directory key file (`~/.ai_daily_brief/key`) for desktop/local dev
// 4. Deterministic fallback salt
func getEncryptionKey(ctx context.Context) []byte {
	// 1. Check Google Cloud Secret Manager
	secretKeyName := os.Getenv("ENCRYPTION_KEY_SECRET_NAME")
	if secretKeyName == "" {
		secretKeyName = defaultKeySecretName
	}

	if smVal, err := GetSecret(ctx, secretKeyName); err == nil && strings.TrimSpace(smVal) != "" {
		hash := sha256.Sum256([]byte(smVal))
		return hash[:]
	}

	// 2. Check environment variable
	secret := os.Getenv("ENCRYPTION_KEY")
	if secret != "" {
		hash := sha256.Sum256([]byte(secret))
		return hash[:]
	}

	// 3. Check user's home directory key file (~/.ai_daily_brief/key) for local runs
	homeDir, err := os.UserHomeDir()
	if err == nil {
		aiKeyDir := filepath.Join(homeDir, ".ai_daily_brief")
		keyPath := filepath.Join(aiKeyDir, "key")

		if _, err := os.Stat(keyPath); os.IsNotExist(err) {
			if err := os.MkdirAll(aiKeyDir, 0700); err == nil {
				newKey := make([]byte, 32)
				if _, err := io.ReadFull(rand.Reader, newKey); err == nil {
					if err := os.WriteFile(keyPath, newKey, 0600); err == nil {
						log.Printf("[Security] Created local symmetric encryption key at %s", keyPath)
					}
				}
			}
		}

		if keyBytes, err := os.ReadFile(keyPath); err == nil && len(keyBytes) > 0 {
			hash := sha256.Sum256(keyBytes)
			return hash[:]
		}
	}

	// 4. Fallback salt
	hash := sha256.Sum256([]byte(defaultSalt))
	return hash[:]
}

// EncryptWithContext encrypts plainText using AES-GCM with context-aware key retrieval
func EncryptWithContext(ctx context.Context, plainText string) (string, error) {
	if plainText == "" {
		return "", nil
	}

	key := getEncryptionKey(ctx)
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

// DecryptWithContext decrypts a hex-encoded cipherText string back to its original plainText
func DecryptWithContext(ctx context.Context, hexCipherText string) (string, error) {
	if hexCipherText == "" {
		return "", nil
	}

	cipherText, err := hex.DecodeString(hexCipherText)
	if err != nil {
		return "", err
	}

	key := getEncryptionKey(ctx)
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

// Encrypt encrypts plainText to a hex-encoded string containing nonce and cipher text
func Encrypt(plainText string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return EncryptWithContext(ctx, plainText)
}

// Decrypt decrypts a hex-encoded cipherText string back to its original plainText
func Decrypt(hexCipherText string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return DecryptWithContext(ctx, hexCipherText)
}
