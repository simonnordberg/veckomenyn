package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Execer captures the Exec method both *pgxpool.Pool and pgx.Tx implement,
// so store helpers can run inside a transaction or directly on the pool.
type Execer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// WeekUpdate describes a partial update to the weeks table. Nil string fields
// are ignored; empty strings clear the column (nullable columns only).
type WeekUpdate struct {
	IsoWeek      *string
	StartDate    *string
	EndDate      *string
	DeliveryDate *string // nullable: "" clears
	OrderDate    *string // nullable: "" clears
	Status       *string
	NotesMD      *string
}

// UpdateWeek applies a partial update to the row identified by id. Returns
// the number of rows updated (0 means not found). Works on either a pool or
// a transaction so callers that need other side-effects in the same unit of
// work can share the tx.
func UpdateWeek(ctx context.Context, db Execer, id int64, u WeekUpdate) (int64, error) {
	var (
		sets []string
		args []any
	)
	add := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	addDate := func(col string, v *string) {
		if v == nil {
			return
		}
		if *v == "" {
			args = append(args, nil)
			sets = append(sets, fmt.Sprintf("%s = NULL", col))
			_ = args // placeholder use suppressed; NULL literal needs no arg
			// Re-slice back: we appended an arg that we don't need.
			args = args[:len(args)-1]
			return
		}
		args = append(args, *v)
		sets = append(sets, fmt.Sprintf("%s = $%d::date", col, len(args)))
	}

	if u.IsoWeek != nil {
		add("iso_week", *u.IsoWeek)
	}
	addDate("start_date", u.StartDate)
	addDate("end_date", u.EndDate)
	addDate("delivery_date", u.DeliveryDate)
	addDate("order_date", u.OrderDate)
	if u.Status != nil {
		add("status", *u.Status)
	}
	if u.NotesMD != nil {
		add("notes_md", *u.NotesMD)
	}
	if len(sets) == 0 {
		return 0, nil
	}
	args = append(args, id)
	query := fmt.Sprintf(`UPDATE weeks SET %s WHERE id = $%d`, strings.Join(sets, ", "), len(args))
	tag, err := db.Exec(ctx, query, args...)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// HouseholdSettings is the singleton defaults row.
type HouseholdSettings struct {
	DefaultDinners         int    `json:"default_dinners"`
	DefaultDeliveryWeekday int    `json:"default_delivery_weekday"` // 1 = Mon ... 7 = Sun
	DefaultOrderOffsetDays int    `json:"default_order_offset_days"`
	DefaultServings        int    `json:"default_servings"`
	Language               string `json:"language"`     // "sv" | "en"
	LLMProvider            string `json:"llm_provider"` // "anthropic" | "openai" | "openai_compat"
	NotesMD                string `json:"notes_md"`
}

func GetHouseholdSettings(ctx context.Context, pool *pgxpool.Pool) (HouseholdSettings, error) {
	var s HouseholdSettings
	err := pool.QueryRow(ctx, `
		SELECT default_dinners, default_delivery_weekday, default_order_offset_days,
		       default_servings, language, llm_provider, notes_md
		FROM household_settings WHERE id = 1`).
		Scan(&s.DefaultDinners, &s.DefaultDeliveryWeekday, &s.DefaultOrderOffsetDays,
			&s.DefaultServings, &s.Language, &s.LLMProvider, &s.NotesMD)
	return s, err
}

// HouseholdSettingsUpdate is a partial update; nil pointers are ignored.
type HouseholdSettingsUpdate struct {
	DefaultDinners         *int
	DefaultDeliveryWeekday *int
	DefaultOrderOffsetDays *int
	DefaultServings        *int
	Language               *string
	LLMProvider            *string
	NotesMD                *string
}

func UpdateHouseholdSettings(ctx context.Context, pool *pgxpool.Pool, u HouseholdSettingsUpdate) error {
	var (
		sets []string
		args []any
	)
	add := func(col string, val any) {
		args = append(args, val)
		sets = append(sets, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	if u.DefaultDinners != nil {
		add("default_dinners", *u.DefaultDinners)
	}
	if u.DefaultDeliveryWeekday != nil {
		add("default_delivery_weekday", *u.DefaultDeliveryWeekday)
	}
	if u.DefaultOrderOffsetDays != nil {
		add("default_order_offset_days", *u.DefaultOrderOffsetDays)
	}
	if u.DefaultServings != nil {
		add("default_servings", *u.DefaultServings)
	}
	if u.Language != nil {
		add("language", *u.Language)
	}
	if u.LLMProvider != nil {
		add("llm_provider", *u.LLMProvider)
	}
	if u.NotesMD != nil {
		add("notes_md", *u.NotesMD)
	}
	if len(sets) == 0 {
		return nil
	}
	query := fmt.Sprintf(`UPDATE household_settings SET %s WHERE id = 1`, strings.Join(sets, ", "))
	_, err := pool.Exec(ctx, query, args...)
	return err
}
