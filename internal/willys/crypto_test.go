package willys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"golang.org/x/crypto/pbkdf2"
)

func TestPKCS7Pad(t *testing.T) {
	tests := []struct {
		input     string
		blockSize int
		wantLen   int
	}{
		{"hello", 16, 16},
		{"1234567890123456", 16, 32}, // exact block size gets full padding block
		{"a", 16, 16},
	}
	for _, tt := range tests {
		padded := pkcs7Pad([]byte(tt.input), tt.blockSize)
		if len(padded) != tt.wantLen {
			t.Errorf("pkcs7Pad(%q, %d) len = %d, want %d", tt.input, tt.blockSize, len(padded), tt.wantLen)
		}
		// Verify padding bytes are correct
		padByte := padded[len(padded)-1]
		for i := len(padded) - int(padByte); i < len(padded); i++ {
			if padded[i] != padByte {
				t.Errorf("padding byte at %d = %d, want %d", i, padded[i], padByte)
			}
		}
	}
}

func TestEncryptCredentialRoundTrip(t *testing.T) {
	plaintext := "test-credential-value"
	result, err := EncryptCredential(plaintext)
	if err != nil {
		t.Fatalf("EncryptCredential: %v", err)
	}

	if len(result.Key) != 16 {
		t.Errorf("key length = %d, want 16", len(result.Key))
	}

	// Verify all digits
	for _, ch := range result.Key {
		if ch < '0' || ch > '9' {
			t.Errorf("key contains non-digit: %c", ch)
		}
	}

	// Decode and decrypt to verify round-trip
	outer, err := base64.StdEncoding.DecodeString(result.Str)
	if err != nil {
		t.Fatalf("decoding outer base64: %v", err)
	}

	parts := strings.SplitN(string(outer), "::", 3)
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts separated by ::, got %d", len(parts))
	}

	iv, err := hex.DecodeString(parts[0])
	if err != nil {
		t.Fatalf("decoding IV hex: %v", err)
	}

	salt, err := hex.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decoding salt hex: %v", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(parts[2])
	if err != nil {
		t.Fatalf("decoding ciphertext base64: %v", err)
	}

	derivedKey := pbkdf2.Key([]byte(result.Key), salt, 1000, 16, sha1.New)
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		t.Fatalf("creating cipher: %v", err)
	}

	decrypted := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(decrypted, ciphertext)

	// Remove PKCS#7 padding
	padLen := int(decrypted[len(decrypted)-1])
	decrypted = decrypted[:len(decrypted)-padLen]

	if string(decrypted) != plaintext {
		t.Errorf("round-trip failed: got %q, want %q", decrypted, plaintext)
	}
}
