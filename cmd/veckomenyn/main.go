package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/simonnordberg/veckomenyn/internal/agent"
	"github.com/simonnordberg/veckomenyn/internal/backup"
	"github.com/simonnordberg/veckomenyn/internal/providers"
	"github.com/simonnordberg/veckomenyn/internal/server"
	"github.com/simonnordberg/veckomenyn/internal/shopping"
	"github.com/simonnordberg/veckomenyn/internal/store"
	"github.com/simonnordberg/veckomenyn/internal/updates"
)

// Set via -ldflags at build time. Defaults are for `go run` / dev builds.
var (
	version = "dev"
	commit  = "unknown"
	builtAt = "unknown"
)

func main() {
	_ = godotenv.Load()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)
	log.Info("veckomenyn starting", "version", version, "commit", commit, "built_at", builtAt)

	addr := envOr("HTTP_ADDR", ":8080")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	snapshotter := openSnapshotter(log, dsn)

	if err := preMigrationBackup(ctx, log, snapshotter, dsn, version); err != nil {
		log.Error("pre-migration snapshot failed; refusing to migrate",
			"err", err,
			"hint", "set VECKOMENYN_SKIP_PREMIGRATION_BACKUP=1 to bypass (dev only)")
		os.Exit(1)
	}

	if err := store.Migrate(ctx, dsn); err != nil {
		log.Error("migrate failed", "err", err)
		os.Exit(1)
	}
	log.Info("migrations applied")

	db, err := store.Open(ctx, dsn)
	if err != nil {
		log.Error("db open failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	masterKey, err := providers.LoadOrGenerateMasterKey(ctx, db.Pool, os.Getenv("MASTER_KEY"), log)
	if err != nil {
		log.Error("master key resolution failed", "err", err)
		os.Exit(1)
	}
	provStore, err := providers.New(db.Pool, masterKey)
	if err != nil {
		log.Error("provider store init failed", "err", err)
		os.Exit(1)
	}
	log.Info("provider secrets encrypted at rest")

	willysShop := shopping.NewWillys(db.Pool, provStore, log)

	ag := agent.New(agent.Config{}, db.Pool, provStore, willysShop, log)

	backupCfg := store.NewBackupConfigStore(db.Pool)
	if snapshotter != nil && snapshotter.CanWrite() {
		go backup.NewScheduler(snapshotter, backupCfg, version, log).Run(ctx)
	}

	var updateChecker *updates.Checker
	if os.Getenv("DISABLE_UPDATE_CHECK") != "1" {
		updateChecker = updates.New("simonnordberg/veckomenyn", version)
	}

	srv := server.New(server.Config{
		Addr:         addr,
		Build:        server.BuildInfo{Version: version, Commit: commit, BuiltAt: builtAt},
		Snapshotter:  snapshotter,
		BackupConfig: backupCfg,
		Updates:      updateChecker,
	}, db, ag, provStore, log)

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-errCh:
		if err != nil {
			log.Error("server error", "err", err)
		}
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("shutdown error", "err", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envOrInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// openSnapshotter resolves a Snapshotter from BACKUP_DIR. Returns nil when
// disabled (env unset). Falls back to a read-only Snapshotter when pg_dump
// isn't on PATH so dev hosts without the postgres client can still list
// existing snapshots through the API.
func openSnapshotter(log *slog.Logger, dsn string) *backup.Snapshotter {
	dir := os.Getenv("BACKUP_DIR")
	if dir == "" {
		log.Warn("BACKUP_DIR not set; backup features disabled. Set BACKUP_DIR to a persistent path to enable.")
		return nil
	}
	snap, err := backup.New(dsn, dir, log)
	if err != nil {
		log.Warn("pg_dump not available; backups read-only (existing dumps listable, no new snapshots)", "err", err)
		ro, openErr := backup.Open(dir, log)
		if openErr != nil {
			log.Error("backup dir init failed", "err", openErr)
			return nil
		}
		return ro
	}
	return snap
}

// preMigrationBackup takes a pg_dump if there are pending migrations and a
// snapshotter is available. Refuses to migrate on dump failure unless
// VECKOMENYN_SKIP_PREMIGRATION_BACKUP=1 is set. Pruning is best-effort.
func preMigrationBackup(ctx context.Context, log *slog.Logger, snap *backup.Snapshotter, dsn, version string) error {
	pending, currentVer, err := store.PendingMigrations(ctx, dsn)
	if err != nil {
		return err
	}
	if pending == 0 {
		return nil
	}

	log.Info("pending migrations detected", "count", pending, "from_version", currentVer)

	if snap == nil {
		log.Warn("no snapshotter configured; pre-migration safety snapshot disabled.")
		return nil
	}

	skip := os.Getenv("VECKOMENYN_SKIP_PREMIGRATION_BACKUP") == "1"
	keep := envOrInt("PREMIGRATION_BACKUP_KEEP", 10)

	if _, err := snap.Snapshot(ctx, backup.ReasonPreMigration, version); err != nil {
		if skip {
			log.Warn("snapshot failed; skip flag set, proceeding", "err", err)
			return nil
		}
		return err
	}

	if removed, err := snap.Prune(backup.ReasonPreMigration, keep); err != nil {
		log.Warn("snapshot prune failed", "err", err)
	} else if len(removed) > 0 {
		log.Info("old snapshots pruned", "removed", len(removed), "keep", keep)
	}
	return nil
}
