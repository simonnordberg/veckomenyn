// willys-import populates the database from markdown preference files.
//
// Each .md file found under --from is inserted as a single row in
// cooking_principles, using the filename (without extension) as the category.
// Re-running the tool upserts on (category), so it's safe to edit a file and
// import again.
//
// Usage:
//
//	DATABASE_URL=postgres://... willys-import --from ./shared-data/preferences
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	from := flag.String("from", "", "directory containing .md preference files")
	flag.Parse()

	if *from == "" {
		log.Fatal("--from is required")
	}
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer conn.Close(ctx)

	files, err := filepath.Glob(filepath.Join(*from, "*.md"))
	if err != nil {
		log.Fatalf("glob: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		log.Fatalf("no .md files found in %s", *from)
	}

	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			log.Fatalf("read %s: %v", f, err)
		}
		category := strings.TrimSuffix(filepath.Base(f), ".md")

		tx, err := conn.Begin(ctx)
		if err != nil {
			log.Fatalf("begin tx: %v", err)
		}
		if _, err := tx.Exec(ctx, "DELETE FROM cooking_principles WHERE category = $1", category); err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("delete %s: %v", category, err)
		}
		if _, err := tx.Exec(ctx, "INSERT INTO cooking_principles (category, body_md) VALUES ($1, $2)", category, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			log.Fatalf("insert %s: %v", category, err)
		}
		if err := tx.Commit(ctx); err != nil {
			log.Fatalf("commit %s: %v", category, err)
		}

		fmt.Printf("imported cooking_principles.%s (%d bytes)\n", category, len(body))
	}
}
