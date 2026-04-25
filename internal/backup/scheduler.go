package backup

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Scheduler runs nightly snapshots in-process. It polls config every minute
// so toggle changes from the UI take effect quickly without restart.
type Scheduler struct {
	snap       *Snapshotter
	config     ConfigReader
	versionTag string
	log        *slog.Logger

	mu     sync.Mutex
	lastAt time.Time
}

// ConfigReader is the minimal surface the scheduler needs from the store.
// Kept narrow so tests can inject a fake without spinning up Postgres.
type ConfigReader interface {
	BackupConfig(ctx context.Context) (Config, error)
}

type Config struct {
	NightlyEnabled bool
	NightlyKeep    int
}

func NewScheduler(snap *Snapshotter, cfg ConfigReader, versionTag string, log *slog.Logger) *Scheduler {
	return &Scheduler{snap: snap, config: cfg, versionTag: versionTag, log: log}
}

// Run blocks until ctx is canceled. Wakes once per minute, takes a snapshot
// when nightly is enabled and 24h has elapsed since the last one.
func (s *Scheduler) Run(ctx context.Context) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	cfg, err := s.config.BackupConfig(ctx)
	if err != nil {
		s.log.Warn("backup config read failed", "err", err)
		return
	}
	if !cfg.NightlyEnabled {
		return
	}

	s.mu.Lock()
	if !s.lastAt.IsZero() && time.Since(s.lastAt) < 24*time.Hour {
		s.mu.Unlock()
		return
	}
	// Cold start: bootstrap lastAt from the most recent nightly on disk so we
	// don't double-up after a restart. Pre-migration and manual snapshots
	// don't count for the 24h cadence.
	if s.lastAt.IsZero() {
		if list, err := s.snap.List(); err == nil {
			for _, b := range list {
				if b.Reason == ReasonNightly {
					s.lastAt = b.CreatedAt
					break
				}
			}
		}
		if !s.lastAt.IsZero() && time.Since(s.lastAt) < 24*time.Hour {
			s.mu.Unlock()
			return
		}
	}
	s.mu.Unlock()

	if _, err := s.snap.Snapshot(ctx, ReasonNightly, s.versionTag); err != nil {
		s.log.Error("nightly snapshot failed", "err", err)
		return
	}
	s.mu.Lock()
	s.lastAt = time.Now()
	s.mu.Unlock()

	if removed, err := s.snap.Prune(ReasonNightly, cfg.NightlyKeep); err != nil {
		s.log.Warn("nightly prune failed", "err", err)
	} else if len(removed) > 0 {
		s.log.Info("nightly snapshots pruned", "removed", len(removed), "keep", cfg.NightlyKeep)
	}
}
