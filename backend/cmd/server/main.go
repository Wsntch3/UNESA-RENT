package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/config"
	"unesarent/backend/internal/db"
	"unesarent/backend/internal/httpapi"
	"unesarent/backend/internal/ktm"
	"unesarent/backend/internal/user"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		slog.Warn("gagal memuat .env", "error", err)
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("konfigurasi tidak valid", "error", err)
		os.Exit(1)
	}

	store, pool, err := buildStore(cfg)
	if err != nil {
		slog.Error("gagal menyiapkan storage", "error", err)
		os.Exit(1)
	}
	if pool != nil {
		defer pool.Close()
	}

	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.TokenTTL)
	svc := user.NewService(store, tokens)

	var ktmHandler *httpapi.KTMHandler
	if pool != nil {
		files, err := ktm.NewLocalStorage(cfg.KTMStorageDir)
		if err != nil {
			slog.Error("gagal menyiapkan penyimpanan KTM", "error", err)
			os.Exit(1)
		}
		ktmSvc := ktm.NewService(ktm.NewPostgresStore(pool), files)
		ktmHandler = httpapi.NewKTMHandler(ktmSvc)
		slog.Info("penyimpanan file KTM", "dir", cfg.KTMStorageDir)
	} else {
		slog.Warn("fitur KTM dinonaktifkan karena DATABASE_URL kosong")
	}

	router := httpapi.NewRouter(cfg, httpapi.NewAuthHandler(svc), ktmHandler, tokens, store)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server berjalan", "addr", cfg.Addr, "token_ttl", cfg.TokenTTL.String())
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server berhenti", "error", err)
			os.Exit(1)
		}
	case <-ctx.Done():
		slog.Info("mematikan server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("gagal shutdown", "error", err)
			os.Exit(1)
		}
	}
}

// buildStore memakai PostgreSQL jika DATABASE_URL diisi, selain itu file JSON.
// Pool dikembalikan agar bisa ditutup saat server berhenti.
func buildStore(cfg config.Config) (user.Store, *pgxpool.Pool, error) {
	if cfg.DatabaseURL == "" {
		memory, err := user.NewMemoryStore(cfg.StorePath)
		if err != nil {
			return nil, nil, err
		}
		slog.Warn("DATABASE_URL kosong, memakai penyimpanan JSON", "path", cfg.StorePath)
		return memory, nil, nil
	}

	pool, err := db.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	slog.Info("penyimpanan user: PostgreSQL")
	return user.NewPostgresStore(pool), pool, nil
}
