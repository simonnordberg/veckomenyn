package shopping

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/simonnordberg/veckomenyn/internal/willys"
)

// DBSessionStore keeps the Willys session in the willys_session singleton
// row. It's the server analogue of willys.FileSessionStore.
type DBSessionStore struct {
	Pool *pgxpool.Pool
}

func NewDBSessionStore(pool *pgxpool.Pool) *DBSessionStore {
	return &DBSessionStore{Pool: pool}
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
	var blob sessionBlob
	if err := json.Unmarshal(raw, &blob); err != nil {
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
	_, err = s.Pool.Exec(ctx, `
		INSERT INTO willys_session (id, cookies_bytea, csrf, refreshed_at)
		VALUES (1, $1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET
			cookies_bytea = EXCLUDED.cookies_bytea,
			csrf = EXCLUDED.csrf,
			refreshed_at = EXCLUDED.refreshed_at`,
		raw, sess.CSRFToken, time.Now())
	return err
}

func (s *DBSessionStore) Clear(ctx context.Context) error {
	_, err := s.Pool.Exec(ctx, `DELETE FROM willys_session WHERE id = 1`)
	return err
}
