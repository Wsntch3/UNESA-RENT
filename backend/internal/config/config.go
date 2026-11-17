package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Addr          string
	CORSOrigins   []string
	JWTSecret     []byte
	TokenTTL      time.Duration
	StorePath     string
	DatabaseURL   string
	KTMStorageDir string
	Env           string // development | production
}

const devSecret = "dev-secret-jangan-dipakai-di-production-ubah-me"

func Load() (Config, error) {
	cfg := Config{
		Addr:        envOr("APP_ADDR", ":8080"),
		TokenTTL:    envDurationOr("JWT_TTL", 24*time.Hour),
		StorePath:   envOr("STORE_PATH", "data/users.json"),
		DatabaseURL: envOr("DATABASE_URL", ""),
		// File KTM disimpan di luar database. Ganti ke Supabase dengan
		// mengimplementasikan ktm.Storage, bukan mengubah path ini.
		KTMStorageDir: envOr("KTM_STORAGE_DIR", "data/ktm"),
		// Default production: lupa mengisi env tidak boleh membuka secret publik.
		Env: envOr("APP_ENV", "production"),
	}

	origins := envOr("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")
	for _, origin := range strings.Split(origins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, origin)
		}
	}

	secret := strings.TrimSpace(os.Getenv("JWT_SECRET"))
	if secret == "" {
		if cfg.Env != "development" {
			return Config{}, fmt.Errorf("JWT_SECRET wajib di-set (atau APP_ENV=development)")
		}
		secret = devSecret
		fmt.Fprintln(os.Stderr, "[WARN] JWT_SECRET kosong, memakai secret development.")
	}
	if len(secret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET minimal 32 karakter")
	}
	cfg.JWTSecret = []byte(secret)

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envDurationOr(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}