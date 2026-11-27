package config

import (
	"fmt"
	"os"
	"strconv"
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

	ListingStorageDir string

	AgreementPDFPath string

	RentalDocumentsDir string
	Env                string

	PublicBaseURL         string
	AllowedStudentDomains []string
	AdminEmails           []string
	AdminLoginOTP         bool

	SMTPHost string
	SMTPPort int
	SMTPUser string
	SMTPPass string
	SMTPFrom string

	BrevoAPIKey string

	OTPPepper             []byte
	OTPTTL                time.Duration
	OTPMaxAttempts        int
	RegistrationTicketTTL time.Duration

	TrustedProxy bool

	StorageDriver      string
	SupabaseURL        string
	SupabaseServiceKey string
	SupabaseBucket     string

	ChatEncKeys          string
	ChatEncActiveVersion int

	CODRequireReturn bool
}

const devSecret = "dev-secret-jangan-dipakai-di-production-ubah-me"

const devChatKey = "ZGV2LWNoYXQtZW5jcnlwdGlvbi1rZXktMzJieXRlcyE="

func Load() (Config, error) {
	cfg := Config{
		Addr:        envAddr(),
		TokenTTL:    envDurationOr("JWT_TTL", 24*time.Hour),
		StorePath:   envOr("STORE_PATH", "data/users.json"),
		DatabaseURL: envOr("DATABASE_URL", ""),

		KTMStorageDir:      envOr("KTM_STORAGE_DIR", "data/ktm"),
		ListingStorageDir:  envOr("LISTING_STORAGE_DIR", "data/listings"),
		AgreementPDFPath:   envOr("AGREEMENT_PDF_PATH", "assets/rental-agreement.pdf"),
		RentalDocumentsDir: envOr("RENTAL_DOCUMENTS_DIR", "data/rental-documents"),

		Env: envOr("APP_ENV", "production"),

		PublicBaseURL:         envOr("PUBLIC_BASE_URL", ""),
		AdminLoginOTP:         envBoolOr("ADMIN_LOGIN_OTP", false),
		SMTPHost:              envOr("SMTP_HOST", ""),
		SMTPPort:              envIntOr("SMTP_PORT", 587),
		SMTPUser:              envOr("SMTP_USER", ""),
		SMTPPass:              envOr("SMTP_PASS", ""),
		SMTPFrom:              envOr("SMTP_FROM", ""),
		BrevoAPIKey:           envOr("BREVO_APIKEY", envOr("BREVO_API_KEY", "")),
		OTPTTL:                envDurationOr("OTP_TTL", 10*time.Minute),
		OTPMaxAttempts:        envIntOr("OTP_MAX_ATTEMPTS", 5),
		RegistrationTicketTTL: envDurationOr("REGISTRATION_TICKET_TTL", 15*time.Minute),
		TrustedProxy:          envBoolOr("TRUSTED_PROXY", false),

		StorageDriver:      envOr("STORAGE_DRIVER", "local"),
		SupabaseURL:        envOr("SUPABASE_URL", ""),
		SupabaseServiceKey: envOr("SUPABASE_SERVICE_ROLE_KEY", ""),
		SupabaseBucket:     envOr("SUPABASE_BUCKET", "unesarent"),

		ChatEncKeys:          envOr("CHAT_ENC_KEYS", ""),
		ChatEncActiveVersion: envIntOr("CHAT_ENC_ACTIVE_VERSION", 1),

		CODRequireReturn: envBoolOr("COD_REQUIRE_RETURN", true),
	}

	origins := envOr("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")
	for _, origin := range strings.Split(origins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, origin)
		}
	}

	cfg.AllowedStudentDomains = splitList(envOr("ALLOWED_STUDENT_EMAIL_DOMAINS", "mhs.unesa.ac.id"))
	cfg.AdminEmails = splitList(envOr("ADMIN_EMAILS", ""))

	if cfg.Env == "production" {

		if _, set := os.LookupEnv("ADMIN_LOGIN_OTP"); !set {
			cfg.AdminLoginOTP = true
		}
		if _, set := os.LookupEnv("TRUSTED_PROXY"); !set {
			cfg.TrustedProxy = true
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

	pepper := strings.TrimSpace(os.Getenv("OTP_PEPPER"))
	if pepper == "" {
		if cfg.Env != "development" {
			return Config{}, fmt.Errorf("OTP_PEPPER wajib di-set (minimal 32 karakter)")
		}
		pepper = "dev-otp-pepper-jangan-dipakai-di-produksi-32"
		fmt.Fprintln(os.Stderr, "[WARN] OTP_PEPPER kosong, memakai pepper development.")
	}
	if len(pepper) < 32 {
		return Config{}, fmt.Errorf("OTP_PEPPER minimal 32 karakter")
	}
	cfg.OTPPepper = []byte(pepper)

	if cfg.Env == "production" {
		if cfg.PublicBaseURL == "" {
			return Config{}, fmt.Errorf("PUBLIC_BASE_URL wajib di-set pada production")
		}
		if cfg.SMTPFrom == "" || (cfg.SMTPHost == "" && cfg.BrevoAPIKey == "") {
			return Config{}, fmt.Errorf("SMTP_FROM wajib di-set dan salah satu SMTP_HOST atau BREVO_APIKEY wajib diisi pada production")
		}
		for _, origin := range cfg.CORSOrigins {
			if strings.Contains(origin, "localhost") || strings.Contains(origin, "127.0.0.1") {
				return Config{}, fmt.Errorf("CORS_ORIGINS tidak boleh berisi localhost pada production")
			}
		}
		if cfg.ChatEncKeys == "" {
			return Config{}, fmt.Errorf("CHAT_ENC_KEYS wajib di-set pada production")
		}
	} else if cfg.ChatEncKeys == "" {
		cfg.ChatEncKeys = fmt.Sprintf(`{"%d":%q}`, cfg.ChatEncActiveVersion, devChatKey)
	}

	if cfg.OTPMaxAttempts < 1 || cfg.OTPMaxAttempts > 10 {
		return Config{}, fmt.Errorf("OTP_MAX_ATTEMPTS harus 1-10")
	}
	if cfg.SMTPPort <= 0 || cfg.SMTPPort > 65535 {
		return Config{}, fmt.Errorf("SMTP_PORT tidak valid")
	}
	if cfg.StorageDriver != "local" && cfg.StorageDriver != "supabase" {
		return Config{}, fmt.Errorf("STORAGE_DRIVER harus local atau supabase")
	}
	if cfg.StorageDriver == "supabase" {
		if cfg.SupabaseURL == "" || cfg.SupabaseServiceKey == "" || cfg.SupabaseBucket == "" {
			return Config{}, fmt.Errorf("SUPABASE_URL, SUPABASE_SERVICE_ROLE_KEY, dan SUPABASE_BUCKET wajib diisi untuk STORAGE_DRIVER=supabase")
		}
	}

	return cfg, nil
}

func splitList(raw string) []string {
	var out []string
	for _, item := range strings.Split(raw, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envAddr() string {
	if addr := strings.TrimSpace(os.Getenv("APP_ADDR")); addr != "" {
		return addr
	}
	if port := strings.TrimSpace(os.Getenv("PORT")); port != "" {
		if strings.HasPrefix(port, ":") {
			return port
		}
		return ":" + port
	}
	return ":8080"
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

func envIntOr(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return v
}

func envBoolOr(key string, fallback bool) bool {
	raw := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if raw == "" {
		return fallback
	}
	switch raw {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}
