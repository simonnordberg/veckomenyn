package willys

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptedCredential holds the encrypted value and the key needed to decrypt it.
type EncryptedCredential struct {
	Key string
	Str string
}

// EncryptCredential encrypts a credential using the same AES-128-CBC scheme
// that the Willys.se frontend uses. The server expects this format.
func EncryptCredential(plaintext string) (EncryptedCredential, error) {
	iv := make([]byte, 16)
	if _, err := rand.Read(iv); err != nil {
		return EncryptedCredential{}, fmt.Errorf("generating IV: %w", err)
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return EncryptedCredential{}, fmt.Errorf("generating salt: %w", err)
	}

	key := randomNumericKey(16)

	derivedKey := pbkdf2.Key([]byte(key), salt, 1000, 16, sha1.New)

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return EncryptedCredential{}, fmt.Errorf("creating cipher: %w", err)
	}

	padded := pkcs7Pad([]byte(plaintext), aes.BlockSize)
	ciphertext := make([]byte, len(padded))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(ciphertext, padded)

	combined := hex.EncodeToString(iv) + "::" + hex.EncodeToString(salt) + "::" + base64.StdEncoding.EncodeToString(ciphertext)
	encoded := base64.StdEncoding.EncodeToString([]byte(combined))

	return EncryptedCredential{Key: key, Str: encoded}, nil
}

func randomNumericKey(length int) string {
	digits := make([]byte, length)
	for i := range digits {
		n, _ := rand.Int(rand.Reader, big.NewInt(10))
		digits[i] = '0' + byte(n.Int64())
	}
	return string(digits)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	pad := make([]byte, padding)
	for i := range pad {
		pad[i] = byte(padding)
	}
	return append(data, pad...)
}
