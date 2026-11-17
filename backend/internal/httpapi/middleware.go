package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/user"
)

type ctxKey string

const (
	userIDKey ctxKey = "user_id"
	roleKey   ctxKey = "user_role"
)

// AuthStateReader dipakai RequireAuth untuk membaca role/status terbaru dari storage.
// user.Store (PostgresStore dan MemoryStore) otomatis memenuhinya.
type AuthStateReader interface {
	FindAuthState(ctx context.Context, id string) (user.AuthState, error)
}

func CORS(allowedOrigins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowed[origin] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
				w.Header().Set("Access-Control-Max-Age", "600")
				w.Header().Add("Vary", "Origin")
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration", time.Since(start).Round(time.Millisecond).String(),
		)
	})
}

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic", "error", rec, "stack", string(debug.Stack()))
				writeError(w, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan pada server")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func RequireAuth(tokens *auth.TokenIssuer, states AuthStateReader) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if header == "" {
				writeError(w, http.StatusUnauthorized, "missing_token", "Header Authorization wajib diisi")
				return
			}

			parts := strings.Fields(header)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				writeError(w, http.StatusUnauthorized, "invalid_token", "Format token harus 'Bearer <token>'")
				return
			}

			claims, err := tokens.Parse(parts[1])
			if err != nil {
				if errors.Is(err, auth.ErrExpiredToken) {
					writeError(w, http.StatusUnauthorized, "token_expired", "Sesi sudah berakhir, silakan login ulang")
					return
				}
				writeError(w, http.StatusUnauthorized, "invalid_token", "Token tidak valid")
				return
			}

			state, err := states.FindAuthState(r.Context(), claims.Subject)
			if err != nil {
				if !errors.Is(err, user.ErrNotFound) {
					slog.Error("gagal membaca status auth", "error", err)
				}
				writeError(w, http.StatusUnauthorized, "invalid_token", "Sesi tidak valid, silakan login ulang")
				return
			}
			if state.Status != user.StatusActive || state.TokenVersion != claims.TokenVersion {
				writeError(w, http.StatusUnauthorized, "invalid_token", "Sesi tidak valid, silakan login ulang")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.Subject)
			ctx = context.WithValue(ctx, roleKey, string(state.Role)) // role dari DB, bukan dari token
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole membatasi endpoint ke role tertentu. Pasang setelah RequireAuth.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := allowed[roleFromContext(r.Context())]; !ok {
				writeError(w, http.StatusForbidden, "forbidden", "Anda tidak memiliki akses ke resource ini")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit membatasi jumlah request per IP dalam satu jendela waktu (in-memory).
// Di belakang reverse proxy, ganti RemoteAddr dengan IP dari header yang tepercaya.
func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	hits := make(map[string]*bucket)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			now := time.Now()

			mu.Lock()
			if len(hits) > 10000 { // cegah memori membengkak
				for k, b := range hits {
					if now.After(b.reset) {
						delete(hits, k)
					}
				}
			}
			b := hits[ip]
			if b == nil || now.After(b.reset) {
				b = &bucket{reset: now.Add(window)}
				hits[ip] = b
			}
			b.count++
			over := b.count > limit
			retry := int(time.Until(b.reset).Seconds()) + 1
			mu.Unlock()

			if over {
				w.Header().Set("Retry-After", strconv.Itoa(retry))
				writeError(w, http.StatusTooManyRequests, "rate_limited", "Terlalu banyak percobaan, coba lagi nanti")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func userIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey).(string)
	return id
}

func roleFromContext(ctx context.Context) string {
	role, _ := ctx.Value(roleKey).(string)
	return role
}