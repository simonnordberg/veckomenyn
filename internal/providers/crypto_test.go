package providers

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
)

func TestParseMasterKey(t *testing.T) {
	t.Run("empty returns nil", func(t *testing.T) {
		b, err := ParseMasterKey("")
		if err != nil || b != nil {
			t.Fatalf("expected (nil, nil), got (%v, %v)", b, err)
		}
	})
	t.Run("whitespace returns nil", func(t *testing.T) {
		b, err := ParseMasterKey("   \n")
		if err != nil || b != nil {
			t.Fatalf("expected (nil, nil), got (%v, %v)", b, err)
		}
	})
	t.Run("standard base64", func(t *testing.T) {
		raw := make([]byte, 32)
		_, _ = rand.Read(raw)
		encoded := base64.StdEncoding.EncodeToString(raw)
		b, err := ParseMasterKey(encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(b) != 32 {
			t.Fatalf("expected 32 bytes, got %d", len(b))
		}
	})
	t.Run("url-safe base64 without padding", func(t *testing.T) {
		raw := make([]byte, 32)
		_, _ = rand.Read(raw)
		// URL-safe without padding triggers the fallback branch.
		encoded := base64.URLEncoding.EncodeToString(raw)
		b, err := ParseMasterKey(encoded)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(b) != 32 {
			t.Fatalf("got %d bytes", len(b))
		}
	})
}

func TestCryptorRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}
	c, err := newCryptor(key)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil {
		t.Fatal("expected non-nil cryptor")
	}

	plain := "sk-ant-apiXX-very-long-secret-value-abcdef"
	wrapped, err := c.encrypt(plain)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !strings.HasPrefix(wrapped, cryptoPrefix) {
		t.Fatalf("missing prefix: %s", wrapped)
	}
	if !isEncrypted(wrapped) {
		t.Fatal("isEncrypted should return true")
	}

	back, err := c.decrypt(wrapped)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if back != plain {
		t.Fatalf("roundtrip mismatch: %q vs %q", back, plain)
	}
}

func TestCryptorWrongKey(t *testing.T) {
	k1 := make([]byte, 32)
	k2 := make([]byte, 32)
	_, _ = rand.Read(k1)
	_, _ = rand.Read(k2)
	c1, _ := newCryptor(k1)
	c2, _ := newCryptor(k2)

	wrapped, err := c1.encrypt("secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c2.decrypt(wrapped); err == nil {
		t.Fatal("expected decrypt with wrong key to fail")
	}
}

func TestCryptorNonceUniqueness(t *testing.T) {
	// Same plaintext under the same key must produce different ciphertexts
	// every time; otherwise a replay attack is trivial.
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	c, _ := newCryptor(key)

	a, _ := c.encrypt("same")
	b, _ := c.encrypt("same")
	if a == b {
		t.Fatal("two encryptions of the same plaintext produced identical output")
	}
}

func TestCryptorRejectsWrongKeyLength(t *testing.T) {
	_, err := newCryptor([]byte("too-short"))
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestCryptorNilForEmptyKey(t *testing.T) {
	c, err := newCryptor(nil)
	if err != nil {
		t.Fatal(err)
	}
	if c != nil {
		t.Fatal("expected nil cryptor for empty key")
	}
}

func TestDecryptPlaintextFails(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	c, _ := newCryptor(key)
	if _, err := c.decrypt("plaintext-value"); err == nil {
		t.Fatal("expected error decrypting plaintext")
	}
}

// The Store exposes EncryptString/DecryptString so callers outside the
// providers package (e.g. the Willys session store) can wrap their own
// blobs with MASTER_KEY without reaching into the private cryptor.

func TestStoreEncryptStringPassthroughWhenNoKey(t *testing.T) {
	s, err := New(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	in := `{"cookies":{"JSESSIONID":"abc"}}`
	out, err := s.EncryptString(in)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if out != in {
		t.Fatalf("expected passthrough when MASTER_KEY is unset; got %q", out)
	}
	back, err := s.DecryptString(in)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if back != in {
		t.Fatalf("expected passthrough, got %q", back)
	}
}

func TestStoreEncryptStringRoundtrip(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	s, err := New(nil, key)
	if err != nil {
		t.Fatal(err)
	}
	plain := `{"cookies":{"JSESSIONID":"abc"},"csrf_token":"xyz"}`
	wrapped, err := s.EncryptString(plain)
	if err != nil {
		t.Fatalf("EncryptString: %v", err)
	}
	if !strings.HasPrefix(wrapped, cryptoPrefix) {
		t.Fatalf("expected enc:v1: prefix, got %q", wrapped)
	}
	back, err := s.DecryptString(wrapped)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if back != plain {
		t.Fatalf("roundtrip mismatch: %q vs %q", back, plain)
	}
}

// A session row written before MASTER_KEY existed has no enc:v1: prefix.
// DecryptString must pass it through unchanged so existing Willys sessions
// keep working when the operator adds a key later.
func TestStoreDecryptStringAcceptsLegacyPlaintext(t *testing.T) {
	key := make([]byte, 32)
	_, _ = rand.Read(key)
	s, err := New(nil, key)
	if err != nil {
		t.Fatal(err)
	}
	legacy := `{"cookies":{"JSESSIONID":"abc"}}`
	back, err := s.DecryptString(legacy)
	if err != nil {
		t.Fatalf("DecryptString: %v", err)
	}
	if back != legacy {
		t.Fatalf("legacy plaintext not passed through: got %q", back)
	}
}
