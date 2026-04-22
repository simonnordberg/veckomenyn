package agent

import "context"

// Week ID lives on the context so cart tools can record their writes to
// the right week without the model having to pass an id every call.

type ctxKey int

const weekIDKey ctxKey = 0

// WithWeekID annotates ctx with the week the agent is working on this
// turn. Zero or negative values are treated as "no week in scope."
func WithWeekID(ctx context.Context, id int64) context.Context {
	if id <= 0 {
		return ctx
	}
	return context.WithValue(ctx, weekIDKey, id)
}

// WeekIDFrom returns the week id set by WithWeekID, or 0 if none.
func WeekIDFrom(ctx context.Context) int64 {
	v, _ := ctx.Value(weekIDKey).(int64)
	return v
}
