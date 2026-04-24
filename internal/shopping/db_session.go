package shopping

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/simonnordberg/veckomenyn/internal/providers"
	"github.com/simonnordberg/veckomenyn/internal/willys"
)

// DBSessionStore keeps the Willys session in the willys_session singleton
// row. It's the server analogue of willys.FileSessionStore.
//
// The session JSON is wrapped with the providers Store's AES-GCM envelope
// when MASTER_KEY is set. That matters because a replayable Willys cookie
// is as sensitive as the stored password; leaving it in cleartext while
// encrypting the password would be inconsistent.
type DBSessionStore struct {
	Pool *pgxpool.Pool
	Prov *providers.Store
}

func NewDBSessionStore(pool *pgxpool.Pool, prov *providers.Store) *DBSessionStore {
	return &DBSessionStore{Pool: pool, Prov: prov}
}

type sessionBlob struct {
	Cookies   map[string]string `json:"cookies"`
	CSRFToken string            `json:"csrf_token"`
}

func (s *DBSessionStore) Load(ctx context.Context) (*willys.Session, error) {
	var raw []byte
	var csrf string
	err := s.Pool.QueryRow(ctx,
		`SELECT cookies_bytea, COALESCE(csrf,'') FROM willys_session WHERE id = 1`).
		Scan(&raw, &csrf)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, willys.ErrNoSession
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, willys.ErrNoSession
	}
	unwrapped, err := s.Prov.DecryptString(string(raw))
	if err != nil {
		// Wrong MASTER_KEY, or value wrapped with a key we no longer have.
		// Treat as no session; the next login re-populates the row.
		return nil, willys.ErrNoSession
	}
	var blob sessionBlob
	if err := json.Unmarshal([]byte(unwrapped), &blob); err != nil {
		return nil, err
	}
	if blob.CSRFToken == "" {
		blob.CSRFToken = csrf
	}
	if len(blob.Cookies) == 0 {
		return nil, willys.ErrNoSession
	}
	return &willys.Session{Cookies: blob.Cookies, CSRFToken: blob.CSRFToken}, nil
}

func (s *DBSessionStore) Save(ctx context.Context, sess willys.Session) error {
	raw, err := json.Marshal(sessionBlob{Cookies: sess.Cookies, CSRFToken: sess.CSRFToken})
	if err != nil {
		return err
	}
	wrapped, err := s.Prov.EncryptString(string(raw))
	if err != nil {
		return err
	}
	// csrf is also persisted as a plaintext column for historic reasons;
	// blank it out so only the encrypted blob carries the token.
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO willys_session (id, cookies_bytea, csrf, refreshed_at)
		VALUES (1, $1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			cookies_bytea = EXCLUDED.cookies_bytea,
			csrf = EXCLUDED.csrf,
			refreshed_at = EXCLUDED.refreshed_at`,
		[]byte(wrapped), "", time.Now())
	return err
}

func (s *DBSessionStore) Clear(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM willys_session WHERE id = 1`)
	return err
}
