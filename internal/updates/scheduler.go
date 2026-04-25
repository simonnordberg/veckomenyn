package updates

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// AutoUpdater runs in-process, checks once an hour whether auto-update is
// enabled and a newer release exists, and fires the trigger if so. Mirrors
// the shape of the backup scheduler so the lifecycle is familiar.
type AutoUpdater struct {
	checker *Checker
	trigger *Trigger
	config  ConfigReader
	log     *slog.Logger

	mu     sync.Mutex
	lastAt time.Time
}

// ConfigReader is the minimal surface the auto-updater needs from the
// store. Kept narrow so tests can inject fakes without a DB.
type ConfigReader interface {
	AutoUpdateEnabled(ctx context.Context) (bool, error)
}

func NewAutoUpdater(checker *Checker, trigger *Trigger, cfg ConfigReader, log *slog.Logger) *AutoUpdater {
	return &AutoUpdater{checker: checker, trigger: trigger, config: cfg, log: log}
}

// Run blocks until ctx is canceled. Wakes hourly. Each tick: read config,
// short-circuit if disabled or trigger unconfigured, query the checker,
// fire the trigger if a newer version is available and at least 23 hours
// have passed since the last fire (so a checker that incorrectly reports
// has_update twice doesn't trigger twice on the same release).
func (a *AutoUpdater) Run(ctx context.Context) {
	t := time.NewTicker(time.Hour)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.tick(ctx)
		}
	}
}

func (a *AutoUpdater) tick(ctx context.Context) {
	if a.checker == nil || a.trigger == nil || !a.trigger.Configured() {
		return
	}
	enabled, err := a.config.AutoUpdateEnabled(ctx)
	if err != nil {
		a.log.Warn("auto-update config read failed", "err", err)
		return
	}
	if !enabled {
		return
	}

	a.mu.Lock()
	if !a.lastAt.IsZero() && time.Since(a.lastAt) < 23*time.Hour {
		a.mu.Unlock()
		return
	}
	a.mu.Unlock()

	status, err := a.checker.Status(ctx)
	if err != nil {
		a.log.Warn("auto-update status check failed", "err", err)
		return
	}
	if !status.HasUpdate {
		return
	}

	a.log.Info("auto-update firing", "current", status.Current, "latest", status.Latest)
	if err := a.trigger.Fire(ctx); err != nil {
		a.log.Error("auto-update trigger failed", "err", err)
		return
	}
	a.mu.Lock()
	a.lastAt = time.Now()
	a.mu.Unlock()
}
