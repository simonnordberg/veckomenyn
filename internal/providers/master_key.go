package providers

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const masterKeyName = "master_key"

// keyChoice enumerates the resolved decision for which master key to use
// based on what's in env vs the DB. Pure logic, easy to test.
type keyChoice int

const (
	choiceUseEnv keyChoice = iota
	choiceUseDB
	choiceMirrorEnvToDB
	choiceGenerate
	choiceMismatchError
)

func decideKey(envSet, dbSet, envEqualsDB bool) keyChoice {
	switch {
	case envSet && dbSet && envEqualsDB:
		return choiceUseEnv
	case envSet && dbSet && !envEqualsDB:
		return choiceMismatchError
	case envSet && !dbSet:
		return choiceMirrorEnvToDB
	case !envSet && dbSet:
		return choiceUseDB
	default:
		return choiceGenerate
	}
}

// LoadOrGenerateMasterKey resolves the master encryption key from env or
// DB, generating and persisting one if neither has it. Behavior:
//
//   - env set, DB empty:    use env, mirror to DB so removing env later
//                           doesn't lose the key.
//   - env set, DB matches:  use env (or DB; identical).
//   - env set, DB differs:  refuse to start. Existing encrypted data
//                           would become unreadable. User must restore
//                           the previous key in env, or wipe and re-enter.
//   - env unset, DB has it: use DB.
//   - both empty:           generate, persist, use.
//
// The DB-stored copy lives in the same DB as the ciphertext it protects.
// That's intentional: losing one already loses the other regardless, and
// it eliminates the openssl-rand-base64 ritual from first-run UX.
func LoadOrGenerateMasterKey(ctx context.Context, pool *pgxpool.Pool, envValue string, log *slog.Logger) ([]byte, error) {
	var dbKey []byte
	err := pool.QueryRow(ctx,
		`SELECT value FROM system_secrets WHERE name = $1`, masterKeyName).
		Scan(&dbKey)
	dbSet := err == nil
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("read master_key from db: %w", err)
	}

	var envKey []byte
	if envValue != "" {
		envKey, err = ParseMasterKey(envValue)
		if err != nil {
			return nil, fmt.Errorf("MASTER_KEY env: %w", err)
		}
		if len(envKey) != 32 {
			return nil, fmt.Errorf("MASTER_KEY env must decode to 32 bytes, got %d", len(envKey))
		}
	}
	envSet := len(envKey) > 0

	switch decideKey(envSet, dbSet, envSet && dbSet && bytes.Equal(envKey, dbKey)) {
	case choiceUseEnv:
		log.Info("master key from env (matches DB-stored copy)")
		return envKey, nil

	case choiceMismatchError:
		return nil, errors.New(
			"MASTER_KEY env does not match the key stored in the database; " +
				"refusing to start because existing encrypted data would become unreadable. " +
				"Either restore the previous key in env, or clear the system_secrets row " +
				"and re-enter all credentials from scratch")

	case choiceMirrorEnvToDB:
		if _, err := pool.Exec(ctx,
			`INSERT INTO system_secrets (name, value) VALUES ($1, $2)`,
			masterKeyName, envKey); err != nil {
			return nil, fmt.Errorf("mirror env master_key to db: %w", err)
		}
		log.Info("master key from env (mirrored to DB so future env-unset still decrypts)")
		return envKey, nil

	case choiceUseDB:
		log.Info("master key loaded from DB")
		return dbKey, nil

	case choiceGenerate:
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("generate master_key: %w", err)
		}
		if _, err := pool.Exec(ctx,
			`INSERT INTO system_secrets (name, value) VALUES ($1, $2)`,
			masterKeyName, key); err != nil {
			return nil, fmt.Errorf("persist generated master_key: %w", err)
		}
		log.Info("master key generated and persisted")
		return key, nil
	}

	return nil, errors.New("unreachable")
}
