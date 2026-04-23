// willys-import-week ingests a historical week from the willys-shopping
// markdown + CSV format into the database:
//   - The Veckoöversikt table determines the dinner list (order = date order).
//   - Each ### heading under "## Recept" becomes one dish with recipe_md.
//   - The CSV's `add, CODE, QTY` lines become cart_items, marked committed.
//   - Any "## Retrospective" section becomes a retrospective row.
//
// Re-running for the same iso_week deletes the prior row first, so imports
// are idempotent.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	mdPath := flag.String("md", "", "path to week markdown file (required)")
	csvPath := flag.String("csv", "", "path to week CSV shopping list (optional)")
	isoWeek := flag.String("iso-week", "", "e.g. 2026-W14 (required)")
	startDate := flag.String("start-date", "", "YYYY-MM-DD, first menu day (required)")
	endDate := flag.String("end-date", "", "YYYY-MM-DD, last menu day (required)")
	orderDate := flag.String("order-date", "", "YYYY-MM-DD, optional")
	deliveryDate := flag.String("delivery-date", "", "YYYY-MM-DD, optional")
	status := flag.String("status", "ordered", "week status (draft|cart_built|ordered)")
	flag.Parse()

	if *mdPath == "" || *isoWeek == "" || *startDate == "" || *endDate == "" {
		log.Fatal("--md, --iso-week, --start-date, --end-date are required")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL required")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer func() { _ = conn.Close(ctx) }()

	md, err := os.ReadFile(*mdPath)
	if err != nil {
		log.Fatal(err)
	}
	parsed := parseMD(string(md))
	if len(parsed.Dinners) == 0 {
		log.Fatal("no dinners parsed from MD")
	}

	var csvItems []cartItem
	if *csvPath != "" {
		b, err := os.ReadFile(*csvPath)
		if err != nil {
			log.Fatal(err)
		}
		csvItems = parseCSV(string(b))
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `DELETE FROM weeks WHERE iso_week = $1`, *isoWeek); err != nil {
		log.Fatalf("delete existing: %v", err)
	}

	var weekID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO weeks (iso_week, start_date, end_date, delivery_date, order_date, status, notes_md)
		VALUES ($1, $2::date, $3::date, NULLIF($4,'')::date, NULLIF($5,'')::date, $6, $7)
		RETURNING id`,
		*isoWeek, *startDate, *endDate, *deliveryDate, *orderDate, *status, parsed.Notes).Scan(&weekID)
	if err != nil {
		log.Fatal(err)
	}

	for i, d := range parsed.Dinners {
		day := addDays(*startDate, i)

		tagsJSON, _ := json.Marshal([]string{})
		sourcingJSON, _ := json.Marshal(d.Sourcing)

		var dishID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO dishes (name, cuisine, recipe_md, servings, tags_json, last_made_at)
			VALUES ($1, NULLIF($2,''), $3, $4, $5::jsonb, $6::date)
			RETURNING id`,
			d.Name, d.Cuisine, d.RecipeMD, d.Servings, string(tagsJSON), day).Scan(&dishID)
		if err != nil {
			log.Fatalf("insert dish %q: %v", d.Name, err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO week_dinners (week_id, day_date, dish_id, servings, sourcing_json, notes, sort_order)
			VALUES ($1, $2::date, $3, $4, $5::jsonb, $6, $7)`,
			weekID, day, dishID, d.Servings, string(sourcingJSON), d.Notes, i); err != nil {
			log.Fatalf("insert dinner: %v", err)
		}
	}

	if parsed.Retrospective != "" {
		if _, err := tx.Exec(ctx,
			`INSERT INTO retrospectives (week_id, notes_md) VALUES ($1, $2)`,
			weekID, parsed.Retrospective); err != nil {
			log.Fatal(err)
		}
	}

	for _, ci := range csvItems {
		if _, err := tx.Exec(ctx, `
			INSERT INTO cart_items (week_id, product_code, qty, reason_md, committed)
			VALUES ($1, $2, $3, $4, true)`,
			weekID, ci.Code, ci.Qty, ci.Reason); err != nil {
			log.Fatal(err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("imported %s: %d dinners, %d cart items, retrospective=%v\n",
		*isoWeek, len(parsed.Dinners), len(csvItems), parsed.Retrospective != "")
}

// ---------------------------------------------------------------------------
// Markdown parsing
// ---------------------------------------------------------------------------

type dinner struct {
	Name     string
	Cuisine  string
	Servings int
	RecipeMD string
	Sourcing map[string]string
	Notes    string
}

type cartItem struct {
	Code   string
	Qty    float64
	Reason string
}

type parsed struct {
	Notes         string
	Dinners       []dinner
	Retrospective string
}

func parseMD(md string) parsed {
	var p parsed

	// Top-level > quote block below the H1 becomes week notes.
	blockRe := regexp.MustCompile(`(?m)^>\s*(.+)$`)
	var notes []string
	for _, m := range blockRe.FindAllStringSubmatch(md, -1) {
		notes = append(notes, strings.TrimSpace(m[1]))
	}
	p.Notes = strings.Join(notes, "\n")

	overview := sliceSection(md, "## Veckoöversikt")
	if overview == "" {
		return p
	}

	var rows []overviewRow
	started := false
	for _, line := range strings.Split(overview, "\n") {
		line = strings.TrimSpace(line)
		if !started {
			if strings.HasPrefix(line, "|-") {
				started = true
			}
			continue
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitMDRow(line)
		if len(cells) < 4 {
			continue
		}
		rows = append(rows, overviewRow{
			Day:      stripEmph(cells[0]),
			Name:     stripEmph(cells[1]),
			Servings: parseInt(stripEmph(cells[2]), 4),
			Sourcing: parseSourcing(stripEmph(cells[3])),
		})
	}

	recipeSection := sliceSection(md, "## Recept")
	var recipes []string
	if recipeSection != "" {
		parts := regexp.MustCompile(`(?m)^### `).Split(recipeSection, -1)
		// parts[0] is the preamble before the first ###, skip.
		for _, chunk := range parts[1:] {
			nl := strings.Index(chunk, "\n")
			if nl < 0 {
				continue
			}
			body := strings.TrimSpace(chunk[nl+1:])
			// Trim trailing section separator "---" if present.
			body = strings.TrimRight(body, "-\n ")
			body = strings.TrimSpace(body)
			recipes = append(recipes, body)
		}
	}

	for i, row := range rows {
		d := dinner{
			Name:     row.Name,
			Servings: row.Servings,
			Sourcing: row.Sourcing,
		}
		if i < len(recipes) {
			d.RecipeMD = recipes[i]
		}
		p.Dinners = append(p.Dinners, d)
	}

	if idx := strings.Index(md, "## Retrospective"); idx >= 0 {
		body := strings.TrimSpace(md[idx+len("## Retrospective"):])
		// Cut at next ## header if any.
		if cut := regexp.MustCompile(`(?m)^## `).FindStringIndex(body); cut != nil {
			body = strings.TrimSpace(body[:cut[0]])
		}
		p.Retrospective = body
	}

	return p
}

type overviewRow struct {
	Day      string
	Name     string
	Servings int
	Sourcing map[string]string
}

// sliceSection returns the text between a heading and the next top-level
// heading or major separator. Returns "" if the heading is absent.
func sliceSection(md, heading string) string {
	start := strings.Index(md, heading)
	if start < 0 {
		return ""
	}
	rest := md[start:]
	// Take until the next ## heading after the first char.
	matches := regexp.MustCompile(`(?m)^## `).FindAllStringIndex(rest, -1)
	if len(matches) > 1 {
		// matches[0] is the heading itself at offset 0; matches[1] is next.
		rest = rest[:matches[1][0]]
	}
	return rest
}

func splitMDRow(line string) []string {
	line = strings.Trim(line, " |")
	cells := strings.Split(line, "|")
	out := make([]string, 0, len(cells))
	for _, c := range cells {
		out = append(out, strings.TrimSpace(c))
	}
	return out
}

func stripEmph(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "*_")
	return strings.TrimSpace(s)
}

func parseInt(s string, def int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return n
	}
	return def
}

func parseSourcing(s string) map[string]string {
	out := map[string]string{}
	s = strings.TrimSpace(s)
	if s == "" || s == "—" || s == "-" {
		return out
	}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if colon := strings.Index(part, ":"); colon > 0 {
			key := strings.ToLower(strings.TrimSpace(part[:colon]))
			val := strings.TrimSpace(part[colon+1:])
			switch key {
			case "fiskhandlare":
				key = "fishmonger"
			case "slaktare":
				key = "butcher"
			case "bageri":
				key = "bakery"
			}
			out[key] = val
		} else {
			out["other"] = part
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// CSV parsing
// ---------------------------------------------------------------------------

func parseCSV(s string) []cartItem {
	var out []cartItem
	for _, raw := range strings.Split(s, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		var reason string
		if hash := strings.Index(line, "#"); hash >= 0 {
			reason = strings.TrimSpace(line[hash+1:])
			line = strings.TrimSpace(line[:hash])
		}
		fields := strings.Split(line, ",")
		if len(fields) < 3 {
			continue
		}
		if strings.TrimSpace(fields[0]) != "add" {
			continue
		}
		code := strings.TrimSpace(fields[1])
		qty, err := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		if err != nil {
			continue
		}
		out = append(out, cartItem{Code: code, Qty: qty, Reason: reason})
	}
	return out
}

func addDays(date string, days int) string {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return date
	}
	return t.AddDate(0, 0, days).Format("2006-01-02")
}
