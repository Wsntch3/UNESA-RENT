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

	"unesarent/backend/internal/admin"
	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/authotp"
	"unesarent/backend/internal/catalog"
	"unesarent/backend/internal/chatcrypto"
	"unesarent/backend/internal/config"
	"unesarent/backend/internal/db"
	"unesarent/backend/internal/email"
	"unesarent/backend/internal/httpapi"
	"unesarent/backend/internal/ktm"
	"unesarent/backend/internal/objectstore"
	"unesarent/backend/internal/rental"
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
	if pool != nil {
		httpapi.SetHealthStore(pool)
	}

	var objects *objectstore.Supabase
	if cfg.StorageDriver == "supabase" {
		objects = objectstore.NewSupabase(cfg.SupabaseURL, cfg.SupabaseServiceKey, cfg.SupabaseBucket)
		slog.Info("penyimpanan objek: Supabase Storage", "bucket", cfg.SupabaseBucket)
	}

	var mailer email.Sender
	if cfg.SMTPHost == "" {
		if cfg.Env != "development" {
			slog.Error("SMTP_HOST wajib di-set pada production")
			os.Exit(1)
		}
		mailer = email.DevSender{}
		slog.Warn("SMTP belum dikonfigurasi, OTP ditulis ke log (development)")
	} else {
		mailer = email.NewSMTPSender(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUser, cfg.SMTPPass, cfg.SMTPFrom)
	}

	var otpStore authotp.Store
	if pool != nil {
		otpStore = authotp.NewPostgresStore(pool)
	} else {
		otpStore = authotp.NewMemoryStore()
		slog.Warn("OTP memakai penyimpanan memori (DATABASE_URL kosong)")
	}
	otpSvc := authotp.NewService(otpStore, mailer, cfg.OTPPepper, cfg.OTPTTL, cfg.OTPMaxAttempts)

	svc := user.NewService(store, tokens, otpSvc, cfg.AllowedStudentDomains, cfg.AdminLoginOTP)

	var ktmHandler *httpapi.KTMHandler
	if pool != nil {
		var files ktm.Storage
		if objects != nil {
			files = ktm.NewSupabaseStorage(objects)
		} else {
			local, err := ktm.NewLocalStorage(cfg.KTMStorageDir)
			if err != nil {
				slog.Error("gagal menyiapkan penyimpanan KTM", "error", err)
				os.Exit(1)
			}
			files = local
			slog.Info("penyimpanan file KTM", "dir", cfg.KTMStorageDir)
		}
		ktmSvc := ktm.NewService(ktm.NewPostgresStore(pool), files)
		ktmHandler = httpapi.NewKTMHandler(ktmSvc)
	} else {
		slog.Warn("fitur KTM dinonaktifkan karena DATABASE_URL kosong")
	}

	var listingHandler *httpapi.ListingHandler
	if pool != nil {
		var files catalog.Storage
		if objects != nil {
			files = catalog.NewSupabaseStorage(objects)
		} else {
			local, err := catalog.NewLocalStorage(cfg.ListingStorageDir)
			if err != nil {
				slog.Error("gagal menyiapkan penyimpanan foto listing", "error", err)
				os.Exit(1)
			}
			files = local
			slog.Info("penyimpanan foto listing", "dir", cfg.ListingStorageDir)
		}
		listingSvc := catalog.NewService(catalog.NewPostgresStore(pool), files)
		listingHandler = httpapi.NewListingHandler(listingSvc)
	} else {
		slog.Warn("fitur katalog listing dinonaktifkan karena DATABASE_URL kosong")
	}

	var adminStore *admin.PostgresStore
	var adminHandler *httpapi.AdminHandler
	if pool != nil {
		adminStore = admin.NewPostgresStore(pool)
		adminHandler = httpapi.NewAdminHandler(admin.NewService(adminStore), cfg.TrustedProxy)
	}

	var rentalHandler *httpapi.RentalHandler
	if pool != nil {
		var files rental.Storage
		if objects != nil {
			files = rental.NewSupabaseStorage(objects)
		} else {
			local, err := rental.NewLocalStorage(cfg.RentalDocumentsDir)
			if err != nil {
				slog.Error("gagal menyiapkan penyimpanan dokumen rental", "error", err)
				os.Exit(1)
			}
			files = local
			slog.Info("penyimpanan dokumen rental", "dir", cfg.RentalDocumentsDir)
		}
		cipher, err := chatcrypto.New(cfg.ChatEncKeys, cfg.ChatEncActiveVersion)
		if err != nil {
			slog.Error("gagal menyiapkan enkripsi chat", "error", err)
			os.Exit(1)
		}
		rentalSvc := rental.NewService(rental.NewPostgresStore(pool, files)).
			WithCrypto(cipher, files).
			WithCODRequireReturn(cfg.CODRequireReturn)
		rentalHandler = httpapi.NewRentalHandler(rentalSvc, cfg.AgreementPDFPath, cfg.CORSOrigins).
			WithAccessPolicy(store, adminStore, cfg.TrustedProxy)
		slog.Info("PDF perjanjian", "path", cfg.AgreementPDFPath)
	} else {
		slog.Warn("fitur rental dinonaktifkan karena DATABASE_URL kosong")
	}

	router := httpapi.NewRouter(cfg, httpapi.NewAuthHandler(svc, cfg.TrustedProxy), ktmHandler, listingHandler, rentalHandler, adminHandler, tokens, store)

	// ReadTimeout/WriteTimeout dimatikan karena koneksi WebSocket berumur panjang.
	// Perlindungan slowloris tetap ada lewat ReadHeaderTimeout.
	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
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
