package crypto

import (
	"bytes"
	"testing"
)

func TestCryptoRoundtrip(t *testing.T) {
	passphrase := "SecureAdminPassphrase2026!#"
	salt, err := GenerateSalt(16)
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}

	key := DeriveKey(passphrase, salt)
	if len(key) != 32 {
		t.Fatalf("Expected key length 32, got %d", len(key))
	}

	original := []byte("BenzCloud Confidential Enterprise Secret Payload")
	encrypted, err := Encrypt(key, original)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(original, decrypted) {
		t.Fatalf("Decrypted content mismatch: got %s, want %s", string(decrypted), string(original))
	}

	// Tampering test
	encrypted[len(encrypted)-1] ^= 0x01
	_, err = Decrypt(key, encrypted)
	if err == nil {
		t.Fatal("Expected error when decrypting tampered data, got nil")
	}
}

func TestGenerateToken(t *testing.T) {
	token1, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if len(token1) != 64 { // 32 bytes in hex = 64 characters
		t.Fatalf("Expected token length 64, got %d", len(token1))
	}

	token2, err := GenerateToken()
	if err != nil {
		t.Fatalf("GenerateToken failed: %v", err)
	}
	if token1 == token2 {
		t.Fatal("Tokens should be unique")
	}
}
