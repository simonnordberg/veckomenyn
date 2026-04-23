package providers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// Encrypted values are stored as "enc:v1:<base64(nonce||ciphertext)>".
// The prefix serves two purposes: it's a sentinel for plaintext/encrypted
// coexistence during rollout, and it's versioned so we can migrate to a
// different scheme later without ambiguity.
const cryptoPrefix = "enc:v1:"

type cryptor struct {
	aead cipher.AEAD
}

// newCryptor builds an AES-256-GCM cryptor from a 32-byte key. Returns nil
// when the key is empty, which callers must handle as "no encryption;
// treat everything as plaintext".
func newCryptor(key []byte) (*cryptor, error) {
	if len(key) == 0 {
		return nil, nil
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("MASTER_KEY must decode to 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &cryptor{aead: aead}, nil
}

// ParseMasterKey decodes a base64 master key. Accepts both standard and
// URL-safe encodings, trims whitespace, and returns (nil, nil) for empty
// input so env-var-absent deployments keep working in plaintext mode.
func ParseMasterKey(s string) ([]byte, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	if b, err := base64.StdEncoding.DecodeString(s); err == nil {
		return b, nil
	}
	return base64.URLEncoding.DecodeString(s)
}

func (c *cryptor) encrypt(plain string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ct := c.aead.Seal(nil, nonce, []byte(plain), nil)
	return cryptoPrefix + base64.StdEncoding.EncodeToString(append(nonce, ct...)), nil
}

func (c *cryptor) decrypt(s string) (string, error) {
	if !strings.HasPrefix(s, cryptoPrefix) {
		return "", errors.New("not an encrypted value")
	}
	blob, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, cryptoPrefix))
	if err != nil {
		return "", err
	}
	if len(blob) < c.aead.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := blob[:c.aead.NonceSize()], blob[c.aead.NonceSize():]
	pt, err := c.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}

func isEncrypted(s string) bool {
	return strings.HasPrefix(s, cryptoPrefix)
}
