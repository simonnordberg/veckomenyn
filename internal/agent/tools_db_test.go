package agent

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"strconv"
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

// seedException inserts a week_exceptions row and returns its id.
func seedException(t *testing.T, pool *pgxpool.Pool, weekID int64, kind, description string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(t.Context(), `
		INSERT INTO week_exceptions (week_id, kind, description)
		VALUES ($1, $2, $3) RETURNING id`,
		weekID, kind, description).Scan(&id)
	if err != nil {
		t.Fatalf("insert exception: %v", err)
	}
	return id
}

func TestUpdateException(t *testing.T) {
	pool := setupTestDB(t)
	ctx := t.Context()

	dish := seedDish(t, pool, "Pasta")
	weekID := seedWeek(t, pool, "2026-W17", "2026-04-21", "2026-04-27", dish)
	excID := seedException(t, pool, weekID, "absence", "Noah away Thu")

	scoped := WithWeekID(ctx, weekID)
	tool := updateExceptionTool(pool)

	t.Run("updates description only", func(t *testing.T) {
		input := json.RawMessage(`{"exception_id": ` + jsonInt(excID) + `, "description": "Noah away Thu-Sun"}`)
		out, err := tool.Call(scoped, input)
		if err != nil {
			t.Fatalf("call: %v", err)
		}
		if !strings.Contains(out, "updated exception") {
			t.Errorf("unexpected result: %q", out)
		}
		var k, d string
		if err := pool.QueryRow(ctx, `SELECT kind, description FROM week_exceptions WHERE id=$1`, excID).Scan(&k, &d); err != nil {
			t.Fatalf("select: %v", err)
		}
		if k != "absence" {
			t.Errorf("kind changed unexpectedly: %q", k)
		}
		if d != "Noah away Thu-Sun" {
			t.Errorf("description not updated: %q", d)
		}
	})

	t.Run("updates kind only", func(t *testing.T) {
		input := json.RawMessage(`{"exception_id": ` + jsonInt(excID) + `, "kind": "other"}`)
		if _, err := tool.Call(scoped, input); err != nil {
			t.Fatalf("call: %v", err)
		}
		var k string
		if err := pool.QueryRow(ctx, `SELECT kind FROM week_exceptions WHERE id=$1`, excID).Scan(&k); err != nil {
			t.Fatalf("select: %v", err)
		}
		if k != "other" {
			t.Errorf("kind not updated: %q", k)
		}
	})

	t.Run("updates both fields", func(t *testing.T) {
		input := json.RawMessage(`{"exception_id": ` + jsonInt(excID) + `, "kind": "absence", "description": "Noah away Fri"}`)
		if _, err := tool.Call(scoped, input); err != nil {
			t.Fatalf("call: %v", err)
		}
		var k, d string
		if err := pool.QueryRow(ctx, `SELECT kind, description FROM week_exceptions WHERE id=$1`, excID).Scan(&k, &d); err != nil {
			t.Fatalf("select: %v", err)
		}
		if k != "absence" || d != "Noah away Fri" {
			t.Errorf("both fields not updated: kind=%q description=%q", k, d)
		}
	})

	t.Run("rejects empty payload", func(t *testing.T) {
		input := json.RawMessage(`{"exception_id": ` + jsonInt(excID) + `}`)
		if _, err := tool.Call(scoped, input); err == nil {
			t.Error("expected error on empty payload")
		}
	})

	t.Run("rejects cross-plan id", func(t *testing.T) {
		dish2 := seedDish(t, pool, "Sallad")
		other := seedWeek(t, pool, "2026-W18", "2026-04-28", "2026-05-04", dish2)
		otherExc := seedException(t, pool, other, "bake", "fika")

		input := json.RawMessage(`{"exception_id": ` + jsonInt(otherExc) + `, "description": "x"}`)
		if _, err := tool.Call(scoped, input); err == nil {
			t.Error("expected refusal on cross-plan exception")
		}
	})

	t.Run("rejects locked plan", func(t *testing.T) {
		if _, err := pool.Exec(ctx, `UPDATE weeks SET status='ordered' WHERE id=$1`, weekID); err != nil {
			t.Fatalf("lock: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, `UPDATE weeks SET status='draft' WHERE id=$1`, weekID)
		})
		input := json.RawMessage(`{"exception_id": ` + jsonInt(excID) + `, "description": "y"}`)
		if _, err := tool.Call(scoped, input); err == nil {
			t.Error("expected refusal on locked plan")
		}
	})
}

func TestDeleteException(t *testing.T) {
	pool := setupTestDB(t)
	ctx := t.Context()

	dish := seedDish(t, pool, "Pasta")
	weekID := seedWeek(t, pool, "2026-W17", "2026-04-21", "2026-04-27", dish)
	excID := seedException(t, pool, weekID, "absence", "Noah away Thu")

	scoped := WithWeekID(ctx, weekID)
	tool := deleteExceptionTool(pool)

	t.Run("deletes on draft plan", func(t *testing.T) {
		input := json.RawMessage(`{"exception_id": ` + jsonInt(excID) + `}`)
		out, err := tool.Call(scoped, input)
		if err != nil {
			t.Fatalf("call: %v", err)
		}
		if !strings.Contains(out, "deleted exception") {
			t.Errorf("unexpected result: %q", out)
		}
		var n int
		if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM week_exceptions WHERE id=$1`, excID).Scan(&n); err != nil {
			t.Fatalf("select: %v", err)
		}
		if n != 0 {
			t.Errorf("expected row gone, count=%d", n)
		}
	})

	t.Run("rejects locked plan", func(t *testing.T) {
		other := seedException(t, pool, weekID, "bake", "fika")
		if _, err := pool.Exec(ctx, `UPDATE weeks SET status='ordered' WHERE id=$1`, weekID); err != nil {
			t.Fatalf("lock: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(ctx, `UPDATE weeks SET status='draft' WHERE id=$1`, weekID)
		})
		input := json.RawMessage(`{"exception_id": ` + jsonInt(other) + `}`)
		if _, err := tool.Call(scoped, input); err == nil {
			t.Error("expected refusal on locked plan")
		}
	})
}

func jsonInt(i int64) string {
	return strconv.FormatInt(i, 10)
}

// Compile-time guard that we're using the same context.Context type as the
// production code; otherwise WithWeekID won't propagate.
var _ context.Context = context.Background()
