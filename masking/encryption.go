// Copyright 2025 Admilson B. F. Cossa
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

// Package masking provides PII masking and field encryption
// Author: Admilson B. F. Cossa

package masking

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"sync"
)

var (
	// ErrInvalidKeyLength indicates the encryption key is not the correct length
	ErrInvalidKeyLength = errors.New("encryption key must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256")
	// ErrEncryptionFailed indicates encryption operation failed
	ErrEncryptionFailed = errors.New("encryption failed")
	// ErrDecryptionFailed indicates decryption operation failed
	ErrDecryptionFailed = errors.New("decryption failed")
	// ErrInvalidCiphertext indicates the ciphertext is malformed
	ErrInvalidCiphertext = errors.New("invalid ciphertext")
)

// FieldEncryptor provides AES-256-GCM field-level encryption
// Thread-safe for concurrent use
type FieldEncryptor struct {
	mu     sync.RWMutex
	key    []byte
	gcm    cipher.AEAD
	prefix string // Prefix for encrypted values (default: "enc:")
}

// NewFieldEncryptor creates a new AES-256 field encryptor
// The key can be any length - it will be hashed to 32 bytes using SHA-256
func NewFieldEncryptor(key string) (*FieldEncryptor, error) {
	if key == "" {
		return nil, ErrInvalidKeyLength
	}

	// Hash the key to ensure 32 bytes for AES-256
	hash := sha256.Sum256([]byte(key))
	keyBytes := hash[:]

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &FieldEncryptor{
		key:    keyBytes,
		gcm:    gcm,
		prefix: "enc:",
	}, nil
}

// NewFieldEncryptorWithKey creates an encryptor with a raw 32-byte key
func NewFieldEncryptorWithKey(key []byte) (*FieldEncryptor, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	// Pad to 32 bytes if needed
	keyBytes := make([]byte, 32)
	copy(keyBytes, key)

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return &FieldEncryptor{
		key:    keyBytes,
		gcm:    gcm,
		prefix: "enc:",
	}, nil
}

// Encrypt encrypts a plaintext value using AES-256-GCM
// Returns base64-encoded ciphertext with prefix
func (e *FieldEncryptor) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Generate random nonce
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", ErrEncryptionFailed
	}

	// Encrypt with GCM (includes authentication)
	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Encode to base64 and add prefix
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return e.prefix + encoded, nil
}

// Decrypt decrypts a base64-encoded ciphertext
// Expects the value to have the encryption prefix
func (e *FieldEncryptor) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check and remove prefix
	if len(ciphertext) <= len(e.prefix) {
		return "", ErrInvalidCiphertext
	}
	if ciphertext[:len(e.prefix)] != e.prefix {
		return "", ErrInvalidCiphertext
	}
	encoded := ciphertext[len(e.prefix):]

	// Decode from base64
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", ErrInvalidCiphertext
	}

	// Extract nonce and ciphertext
	nonceSize := e.gcm.NonceSize()
	if len(data) < nonceSize {
		return "", ErrInvalidCiphertext
	}

	nonce, encryptedData := data[:nonceSize], data[nonceSize:]

	// Decrypt with authentication verification
	plaintext, err := e.gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", ErrDecryptionFailed
	}

	return string(plaintext), nil
}

// IsEncrypted checks if a value appears to be encrypted (has prefix)
func (e *FieldEncryptor) IsEncrypted(value string) bool {
	return len(value) > len(e.prefix) && value[:len(e.prefix)] == e.prefix
}

// SetPrefix sets a custom prefix for encrypted values
func (e *FieldEncryptor) SetPrefix(prefix string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.prefix = prefix
}

// GlobalFieldEncryptor is the singleton encryptor for system-wide use
var GlobalFieldEncryptor *FieldEncryptor

// InitGlobalEncryptor initializes the global field encryptor
func InitGlobalEncryptor(key string) error {
	encryptor, err := NewFieldEncryptor(key)
	if err != nil {
		return err
	}
	GlobalFieldEncryptor = encryptor
	return nil
}
