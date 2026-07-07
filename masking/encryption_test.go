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
	"strings"
	"testing"
)

func TestNewFieldEncryptor(t *testing.T) {
	t.Run("valid key", func(t *testing.T) {
		enc, err := NewFieldEncryptor("my-secret-key-12345")
		if err != nil {
			t.Fatalf("NewFieldEncryptor failed: %v", err)
		}
		if enc == nil {
			t.Fatal("Expected non-nil encryptor")
		}
	})

	t.Run("empty key", func(t *testing.T) {
		_, err := NewFieldEncryptor("")
		if err != ErrInvalidKeyLength {
			t.Errorf("Expected ErrInvalidKeyLength, got %v", err)
		}
	})
}

func TestNewFieldEncryptorWithKey(t *testing.T) {
	t.Run("32 byte key", func(t *testing.T) {
		key := make([]byte, 32)
		for i := range key {
			key[i] = byte(i)
		}
		enc, err := NewFieldEncryptorWithKey(key)
		if err != nil {
			t.Fatalf("NewFieldEncryptorWithKey failed: %v", err)
		}
		if enc == nil {
			t.Fatal("Expected non-nil encryptor")
		}
	})

	t.Run("24 byte key", func(t *testing.T) {
		key := make([]byte, 24)
		_, err := NewFieldEncryptorWithKey(key)
		if err != nil {
			t.Fatalf("Expected success with 24-byte key: %v", err)
		}
	})

	t.Run("16 byte key", func(t *testing.T) {
		key := make([]byte, 16)
		_, err := NewFieldEncryptorWithKey(key)
		if err != nil {
			t.Fatalf("Expected success with 16-byte key: %v", err)
		}
	})

	t.Run("invalid key length", func(t *testing.T) {
		key := make([]byte, 15) // Invalid length
		_, err := NewFieldEncryptorWithKey(key)
		if err != ErrInvalidKeyLength {
			t.Errorf("Expected ErrInvalidKeyLength, got %v", err)
		}
	})
}

// TestNewFieldEncryptorWithKey_NoZeroPadding proves the key material is used
// verbatim (no silent zero-padding of a 16/24-byte key up to a 32-byte
// AES-256 key). If the implementation zero-padded a 16-byte key to 32 bytes,
// then an AES-256 encryptor built from that same key padded to 32 bytes would
// share the effective key and be able to decrypt the 16-byte encryptor's
// ciphertext. It must NOT.
func TestNewFieldEncryptorWithKey_NoZeroPadding(t *testing.T) {
	key16 := make([]byte, 16)
	for i := range key16 {
		key16[i] = byte(i + 1) // non-zero material
	}

	// Real AES-128 encryptor using the 16-byte key verbatim.
	enc128, err := NewFieldEncryptorWithKey(key16)
	if err != nil {
		t.Fatalf("16-byte key setup failed: %v", err)
	}

	const secret = "sensitive-value-123"
	ct, err := enc128.Encrypt(secret)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	// Round-trip through the public interface with the true key.
	pt, err := enc128.Decrypt(ct)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if pt != secret {
		t.Fatalf("round-trip mismatch: got %q want %q", pt, secret)
	}

	// Build an AES-256 encryptor from the SAME 16 bytes zero-padded to 32.
	// This is exactly what the old buggy implementation used internally. It
	// must NOT be able to decrypt the AES-128 ciphertext, confirming the two
	// encryptors do not share an effective key (i.e. no zero-padding occurred).
	padded := make([]byte, 32)
	copy(padded, key16)
	enc256, err := NewFieldEncryptorWithKey(padded)
	if err != nil {
		t.Fatalf("32-byte key setup failed: %v", err)
	}
	if _, err := enc256.Decrypt(ct); err == nil {
		t.Fatal("AES-256 zero-padded key decrypted AES-128 ciphertext; key was silently padded")
	}
}

// TestNewFieldEncryptorWithKey_RoundTripAllVariants verifies encrypt->decrypt
// round-trips through the public interface for every supported key length.
func TestNewFieldEncryptorWithKey_RoundTripAllVariants(t *testing.T) {
	for _, size := range []int{16, 24, 32} {
		key := make([]byte, size)
		for i := range key {
			key[i] = byte(i*7 + 3)
		}
		enc, err := NewFieldEncryptorWithKey(key)
		if err != nil {
			t.Fatalf("%d-byte key setup failed: %v", size, err)
		}
		const msg = "round-trip-payload"
		ct, err := enc.Encrypt(msg)
		if err != nil {
			t.Fatalf("%d-byte encrypt failed: %v", size, err)
		}
		pt, err := enc.Decrypt(ct)
		if err != nil {
			t.Fatalf("%d-byte decrypt failed: %v", size, err)
		}
		if pt != msg {
			t.Fatalf("%d-byte round-trip mismatch: got %q want %q", size, pt, msg)
		}
	}
}

func TestFieldEncryptorEncryptDecrypt(t *testing.T) {
	enc, err := NewFieldEncryptor("test-encryption-key")
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"simple text", "Hello, World!"},
		{"email", "user@example.com"},
		{"SSN", "123-45-6789"},
		{"credit card", "4111-1111-1111-1111"},
		{"unicode", "こんにちは世界"},
		{"empty", ""},
		{"long text", strings.Repeat("a", 10000)},
		{"special chars", `!@#$%^&*()_+-=[]{}|;':",.<>?/~` + "`"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := enc.Encrypt(tc.plaintext)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Verify encrypted value has prefix (except for empty)
			if tc.plaintext != "" {
				if !enc.IsEncrypted(encrypted) {
					t.Errorf("Encrypted value should have prefix")
				}
			}

			decrypted, err := enc.Decrypt(encrypted)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("Decrypt mismatch: got %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestFieldEncryptorDifferentCiphertexts(t *testing.T) {
	enc, _ := NewFieldEncryptor("test-key")
	plaintext := "same-input"

	encrypted1, _ := enc.Encrypt(plaintext)
	encrypted2, _ := enc.Encrypt(plaintext)

	// Due to random nonce, same plaintext should produce different ciphertexts
	if encrypted1 == encrypted2 {
		t.Error("Same plaintext should produce different ciphertexts (random nonce)")
	}

	// But both should decrypt to the same value
	decrypted1, _ := enc.Decrypt(encrypted1)
	decrypted2, _ := enc.Decrypt(encrypted2)

	if decrypted1 != decrypted2 || decrypted1 != plaintext {
		t.Error("Both ciphertexts should decrypt to original plaintext")
	}
}

func TestFieldEncryptorDecryptErrors(t *testing.T) {
	enc, _ := NewFieldEncryptor("test-key")

	t.Run("invalid prefix", func(t *testing.T) {
		_, err := enc.Decrypt("notenc:someciphertext")
		if err != ErrInvalidCiphertext {
			t.Errorf("Expected ErrInvalidCiphertext, got %v", err)
		}
	})

	t.Run("invalid base64", func(t *testing.T) {
		_, err := enc.Decrypt("enc:not-valid-base64!!!")
		if err != ErrInvalidCiphertext {
			t.Errorf("Expected ErrInvalidCiphertext, got %v", err)
		}
	})

	t.Run("too short", func(t *testing.T) {
		_, err := enc.Decrypt("enc:")
		if err != ErrInvalidCiphertext {
			t.Errorf("Expected ErrInvalidCiphertext, got %v", err)
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		encrypted, _ := enc.Encrypt("test")
		// Tamper with the ciphertext
		tampered := encrypted[:len(encrypted)-5] + "XXXXX"
		_, err := enc.Decrypt(tampered)
		if err == nil {
			t.Error("Expected decryption to fail on tampered ciphertext")
		}
	})
}

func TestFieldEncryptorWrongKey(t *testing.T) {
	enc1, _ := NewFieldEncryptor("key-one")
	enc2, _ := NewFieldEncryptor("key-two")

	encrypted, _ := enc1.Encrypt("secret data")

	// Decrypting with different key should fail
	_, err := enc2.Decrypt(encrypted)
	if err == nil {
		t.Error("Expected decryption to fail with wrong key")
	}
}

func TestFieldEncryptorSetPrefix(t *testing.T) {
	enc, _ := NewFieldEncryptor("test-key")
	enc.SetPrefix("encrypted:")

	encrypted, _ := enc.Encrypt("test")
	if !strings.HasPrefix(encrypted, "encrypted:") {
		t.Errorf("Expected prefix 'encrypted:', got %s", encrypted)
	}

	if !enc.IsEncrypted(encrypted) {
		t.Error("IsEncrypted should return true")
	}
}

func TestFieldEncryptorConcurrency(t *testing.T) {
	enc, _ := NewFieldEncryptor("concurrent-test-key")

	const goroutines = 100
	const iterations = 100

	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < iterations; j++ {
				plaintext := "test-data"
				encrypted, err := enc.Encrypt(plaintext)
				if err != nil {
					t.Errorf("Goroutine %d: Encrypt failed: %v", id, err)
					done <- false
					return
				}
				decrypted, err := enc.Decrypt(encrypted)
				if err != nil {
					t.Errorf("Goroutine %d: Decrypt failed: %v", id, err)
					done <- false
					return
				}
				if decrypted != plaintext {
					t.Errorf("Goroutine %d: Mismatch", id)
					done <- false
					return
				}
			}
			done <- true
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		if !<-done {
			t.Fatal("Concurrency test failed")
		}
	}
}

func TestGlobalFieldEncryptor(t *testing.T) {
	err := InitGlobalEncryptor("global-test-key")
	if err != nil {
		t.Fatalf("InitGlobalEncryptor failed: %v", err)
	}

	if GlobalFieldEncryptor == nil {
		t.Fatal("GlobalFieldEncryptor should be initialized")
	}

	// Test it works
	encrypted, err := GlobalFieldEncryptor.Encrypt("test")
	if err != nil {
		t.Fatalf("Global encrypt failed: %v", err)
	}

	decrypted, err := GlobalFieldEncryptor.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Global decrypt failed: %v", err)
	}

	if decrypted != "test" {
		t.Error("Global encryptor mismatch")
	}
}

func BenchmarkFieldEncryptor_Encrypt(b *testing.B) {
	enc, _ := NewFieldEncryptor("benchmark-key")
	plaintext := "user@example.com"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = enc.Encrypt(plaintext)
	}
}

func BenchmarkFieldEncryptor_Decrypt(b *testing.B) {
	enc, _ := NewFieldEncryptor("benchmark-key")
	encrypted, _ := enc.Encrypt("user@example.com")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = enc.Decrypt(encrypted)
	}
}
