package agent

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Week ID lives on the context so tools can record their writes to the
// right plan without the model having to pass an id every call.

type ctxKey int

const weekIDKey ctxKey = 0

// WithWeekID annotates ctx with the plan the agent is working on this
// turn. Zero or negative values are treated as "no plan in scope."
func WithWeekID(ctx context.Context, id int64) context.Context {
	if id <= 0 {
		return ctx
	}
	return context.WithValue(ctx, weekIDKey, id)
}

// WeekIDFrom returns the plan id set by WithWeekID, or 0 if none.
func WeekIDFrom(ctx context.Context) int64 {
	v, _ := ctx.Value(weekIDKey).(int64)
	return v
}

// ResolvePlan returns the plan id that a write tool should target. When the
// chat is bound to a plan, that plan wins: the model can omit week_id and we
// fill it in; an explicit week_id that doesn't match is refused so one plan's
// chat can't quietly edit a different plan. When no plan is in scope the
// caller's explicit id is used.
func ResolvePlan(ctx context.Context, provided int64) (int64, error) {
	curr := WeekIDFrom(ctx)
	if curr > 0 {
		if provided > 0 && provided != curr {
			return 0, fmt.Errorf("this chat is tied to plan id=%d; refuse to edit plan id=%d. Ask the user to open that plan first", curr, provided)
		}
		return curr, nil
	}
	if provided > 0 {
		return provided, nil
	}
	return 0, fmt.Errorf("no plan in scope; pass week_id or open a plan first")
}

// rowQueryer is the subset of pgx.Tx / *pgxpool.Pool used for the scope check.
type rowQueryer interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// CheckDinnerInScope refuses dinner-level writes when the dinner belongs to
// a different plan than the one this chat is bound to. If no plan is in
// scope, no constraint is applied.
func CheckDinnerInScope(ctx context.Context, db rowQueryer, dinnerID int64) error {
	curr := WeekIDFrom(ctx)
	if curr == 0 {
		return nil
	}
	var weekID int64
	if err := db.QueryRow(ctx, `SELECT week_id FROM week_dinners WHERE id = $1`, dinnerID).Scan(&weekID); err != nil {
		return err
	}
	if weekID != curr {
		return fmt.Errorf("dinner id=%d belongs to plan id=%d; this chat is tied to plan id=%d. Ask the user to open that plan to edit it", dinnerID, weekID, curr)
	}
	return nil
}
