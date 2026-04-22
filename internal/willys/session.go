package willys

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// Session is what the client persists between calls.
type Session struct {
	Cookies   map[string]string `json:"cookies"`
	CSRFToken string            `json:"csrfToken"`
}

// SessionStore abstracts where the session lives. The CLI uses a file on
// disk; the server stores it in Postgres so credentials and session cookies
// all sit together.
type SessionStore interface {
	Load(ctx context.Context) (*Session, error)
	Save(ctx context.Context, s Session) error
	Clear(ctx context.Context) error
}

// ErrNoSession signals an empty store. Callers should treat it as "log in
// first", not as a hard error.
var ErrNoSession = errors.New("no saved session")

// ---------------------------------------------------------------------------
// Default: disk-backed store (used by the CLI)
// ---------------------------------------------------------------------------

type FileSessionStore struct {
	Path string
}

func DefaultFileStore() *FileSessionStore {
	dir, _ := os.UserConfigDir()
	return &FileSessionStore{Path: filepath.Join(dir, "willys-cli", "session.json")}
}

func (f *FileSessionStore) Load(_ context.Context) (*Session, error) {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoSession
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Cookies == nil {
		return nil, ErrNoSession
	}
	return &s, nil
}

func (f *FileSessionStore) Save(_ context.Context, s Session) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(f.Path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(f.Path, data, 0o600)
}

func (f *FileSessionStore) Clear(_ context.Context) error {
	err := os.Remove(f.Path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
