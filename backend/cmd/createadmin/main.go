package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/config"
	"unesarent/backend/internal/db"
)

func main() {
	email := flag.String("email", os.Getenv("ADMIN_EMAIL"), "email admin")
	password := flag.String("password", os.Getenv("ADMIN_PASSWORD"), "password admin (min 12 karakter)")
	name := flag.String("name", "Administrator UNESA Rent", "nama admin")
	flag.Parse()

	if !strings.Contains(*email, "@") || len(*password) < 12 {
		log.Fatal("email valid dan password minimal 12 karakter wajib diisi")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	hash, err := auth.HashPassword(*password)
	if err != nil {
		log.Fatal(err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO users (name, email, password_hash, role, status, email_verified_at)
		VALUES ($1, lower($2), $3, 'admin_dev', 'active', now())
		ON CONFLICT (lower(email)) DO UPDATE
		   SET name = EXCLUDED.name,
		       password_hash = EXCLUDED.password_hash,
		       role = 'admin_dev',
		       status = 'active',
		       email_verified_at = now(),
		       token_version = users.token_version + 1,
		       updated_at = now()
	`, *name, *email, hash)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("admin siap:", strings.ToLower(*email))
}
