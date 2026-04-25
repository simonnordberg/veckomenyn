package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/simonnordberg/veckomenyn/internal/migrations"
)

// setupTestDB opens a connection to TEST_DATABASE_URL, runs migrations,
// and truncates the tables touched by these tests. Skips if the env var
// is unset so `go test` works on hosts without a Postgres available.
//
// The test DB shares schema with the main app's DB; do not point this at
// production. A throwaway compose db is the intended target:
//
//	TEST_DATABASE_URL=postgres://veckomenyn:veckomenyn@localhost:5432/veckomenyn?sslmode=disable go test ./internal/agent/...
func setupTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := t.Context()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open sql: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("set dialect: %v", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	// Wipe the tables we touch so each test starts clean. RESTART IDENTITY
	// keeps autoincrement starts predictable across test runs.
	if _, err := pool.Exec(ctx, `
		TRUNCATE dish_ratings, week_dinners, weeks, dishes
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	return pool
}

// seedDish inserts a dish and returns its id.
func seedDish(t *testing.T, pool *pgxpool.Pool, name string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(t.Context(),
		`INSERT INTO dishes (name) VALUES ($1) RETURNING id`, name).Scan(&id)
	if err != nil {
		t.Fatalf("insert dish %q: %v", name, err)
	}
	return id
}

// seedWeek inserts a week and a dinner referencing the given dish on the
// week's start date. Returns the week id.
func seedWeek(t *testing.T, pool *pgxpool.Pool, isoWeek, start, end string, dishID int64) int64 {
	t.Helper()
	var weekID int64
	err := pool.QueryRow(t.Context(), `
		INSERT INTO weeks (iso_week, start_date, end_date)
		VALUES ($1, $2::date, $3::date)
		RETURNING id`, isoWeek, start, end).Scan(&weekID)
	if err != nil {
		t.Fatalf("insert week %q: %v", isoWeek, err)
	}
	if _, err := pool.Exec(t.Context(), `
		INSERT INTO week_dinners (week_id, day_date, dish_id)
		VALUES ($1, $2::date, $3)`, weekID, start, dishID); err != nil {
		t.Fatalf("insert week_dinner: %v", err)
	}
	return weekID
}

func TestListDishesRecent_excludesCurrentWeek(t *testing.T) {
	pool := setupTestDB(t)
	ctx := t.Context()

	pastDish := seedDish(t, pool, "Pasta carbonara")
	currentDish := seedDish(t, pool, "Tikka masala")

	// Past week: 2 weeks ago, fully eaten.
	seedWeek(t, pool, "2026-W15", "2026-04-06", "2026-04-12", pastDish)
	// Current week: this is the plan being edited.
	currentWeekID := seedWeek(t, pool, "2026-W17", "2026-04-21", "2026-04-27", currentDish)

	tool := listDishesRecentTool(pool)
	input := json.RawMessage(`{"weeks": 4}`)

	t.Run("no scope returns both weeks", func(t *testing.T) {
		out, err := tool.Call(ctx, input)
		if err != nil {
			t.Fatalf("call: %v", err)
		}
		if !strings.Contains(out, "Pasta carbonara") {
			t.Errorf("expected past dish in output, got %q", out)
		}
		if !strings.Contains(out, "Tikka masala") {
			t.Errorf("expected current dish in output (no scope), got %q", out)
		}
	})

	t.Run("scoped to current week excludes that week", func(t *testing.T) {
		scoped := WithWeekID(ctx, currentWeekID)
		out, err := tool.Call(scoped, input)
		if err != nil {
			t.Fatalf("call: %v", err)
		}
		if !strings.Contains(out, "Pasta carbonara") {
			t.Errorf("expected past dish in output, got %q", out)
		}
		if strings.Contains(out, "Tikka masala") {
			t.Errorf("current-week dish leaked into scoped output: %q", out)
		}
	})
}

func TestSearchHistory_excludesCurrentWeekDishes(t *testing.T) {
	pool := setupTestDB(t)
	ctx := t.Context()

	pastDish := seedDish(t, pool, "Tacos al pastor")
	currentDish := seedDish(t, pool, "Tacos suecos")

	seedWeek(t, pool, "2026-W14", "2026-03-30", "2026-04-05", pastDish)
	currentWeekID := seedWeek(t, pool, "2026-W17", "2026-04-21", "2026-04-27", currentDish)

	tool := searchHistoryTool(pool)
	input := json.RawMessage(`{"query": "tacos", "kind": "dishes"}`)

	t.Run("no scope returns both", func(t *testing.T) {
		out, err := tool.Call(ctx, input)
		if err != nil {
			t.Fatalf("call: %v", err)
		}
		if !strings.Contains(out, "Tacos al pastor") || !strings.Contains(out, "Tacos suecos") {
			t.Errorf("expected both dishes in output, got %q", out)
		}
	})

	t.Run("scoped excludes current week's dish", func(t *testing.T) {
		scoped := WithWeekID(ctx, currentWeekID)
		out, err := tool.Call(scoped, input)
		if err != nil {
			t.Fatalf("call: %v", err)
		}
		if !strings.Contains(out, "Tacos al pastor") {
			t.Errorf("expected past dish in output, got %q", out)
		}
		if strings.Contains(out, "Tacos suecos") {
			t.Errorf("current-week dish leaked into scoped search: %q", out)
		}
	})
}

// Compile-time guard that we're using the same context.Context type as the
// production code; otherwise WithWeekID won't propagate.
var _ context.Context = context.Background()
