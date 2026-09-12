package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// PBKDF2Iterations defines the mandatory 100,000 rounds of PBKDF2 key derivation.
	PBKDF2Iterations = 100000
	// KeyLength is 32 bytes for AES-256.
	KeyLength = 32
	// SaltLength is 16 bytes minimum.
	SaltLength = 16
)

var (
	ErrCiphertextTooShort = errors.New("ciphertext too short")
	ErrDecryptionFailed   = errors.New("decryption failed or corrupted data")
)

// DeriveKey derives a 32-byte AES-256 key from a passphrase and salt using PBKDF2 with 100,000 iterations.
func DeriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, PBKDF2Iterations, KeyLength, sha256.New)
}

// GenerateSalt creates a cryptographically secure random salt of the specified length.
func GenerateSalt(length int) ([]byte, error) {
	if length < SaltLength {
		length = SaltLength
	}
	salt := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate random salt: %w", err)
	}
	return salt, nil
}

// GenerateToken generates a cryptographically random 32-byte hex token.
func GenerateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a random 12-byte nonce.
// The output format is: [nonce (12 bytes)] + [ciphertext + 16-byte auth tag].
func Encrypt(key, plaintext []byte) ([]byte, error) {
	if len(key) != KeyLength {
		return nil, fmt.Errorf("invalid key length: must be %d bytes", KeyLength)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts AES-256-GCM ciphertext created by Encrypt.
func Decrypt(key, ciphertext []byte) ([]byte, error) {
	if len(key) != KeyLength {
		return nil, fmt.Errorf("invalid key length: must be %d bytes", KeyLength)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher block: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, actualCiphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}
	return plaintext, nil
}
