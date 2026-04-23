package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/simonnordberg/veckomenyn/internal/agent"
	"github.com/simonnordberg/veckomenyn/internal/providers"
	"github.com/simonnordberg/veckomenyn/internal/server"
	"github.com/simonnordberg/veckomenyn/internal/shopping"
	"github.com/simonnordberg/veckomenyn/internal/store"
)

func main() {
	_ = godotenv.Load()

	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	addr := envOr("HTTP_ADDR", ":8080")
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

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

	masterKey, err := providers.ParseMasterKey(os.Getenv("MASTER_KEY"))
	if err != nil {
		log.Error("invalid MASTER_KEY", "err", err)
		os.Exit(1)
	}
	provStore, err := providers.New(db.Pool, masterKey)
	if err != nil {
		log.Error("provider store init failed", "err", err)
		os.Exit(1)
	}
	if provStore.HasEncryption() {
		log.Info("provider secrets encrypted at rest")
	} else {
		log.Warn("MASTER_KEY not set; provider secrets stored in cleartext. Set a 32-byte base64 key to enable encryption.")
	}

	willysShop := shopping.NewWillys(db.Pool, provStore, log)

	ag := agent.New(agent.Config{}, db.Pool, provStore, willysShop, log)

	srv := server.New(server.Config{Addr: addr}, db, ag, provStore, log)

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
