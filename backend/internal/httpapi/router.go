package httpapi

import (
	"net/http"
	"time"

	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/config"
	"unesarent/backend/internal/user"
)

func NewRouter(cfg config.Config, authHandler *AuthHandler, ktmHandler *KTMHandler, tokens *auth.TokenIssuer, states AuthStateReader) http.Handler {
	mux := http.NewServeMux()
	limit := RateLimit(10, time.Minute)

	mux.HandleFunc("GET /api/health", Health)
	mux.Handle("POST /api/auth/register", limit(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /api/auth/login", limit(http.HandlerFunc(authHandler.Login)))
	mux.Handle("GET /api/auth/me", RequireAuth(tokens, states)(http.HandlerFunc(authHandler.Me)))

	if ktmHandler != nil {
		requireAuth := RequireAuth(tokens, states)
		requireMahasiswa := RequireRole(string(user.RoleMahasiswa))
		requireAdmin := RequireRole(string(user.RoleAdminDev))
		ktmLimit := RateLimit(20, time.Minute)

		mux.Handle("POST /api/ktm", ktmLimit(requireAuth(requireMahasiswa(http.HandlerFunc(ktmHandler.Submit)))))
		mux.Handle("GET /api/ktm/me", requireAuth(requireMahasiswa(http.HandlerFunc(ktmHandler.Mine))))
		mux.Handle("GET /api/ktm/{id}/file", requireAuth(http.HandlerFunc(ktmHandler.File)))

		mux.Handle("GET /api/admin/ktm", requireAuth(requireAdmin(http.HandlerFunc(ktmHandler.AdminList))))
		mux.Handle("POST /api/admin/ktm/{id}/verify", requireAuth(requireAdmin(http.HandlerFunc(ktmHandler.Verify))))
		mux.Handle("POST /api/admin/ktm/{id}/reject", requireAuth(requireAdmin(http.HandlerFunc(ktmHandler.Reject))))
		mux.Handle("POST /api/admin/ktm/{id}/revoke", requireAuth(requireAdmin(http.HandlerFunc(ktmHandler.Revoke))))
	}

	return CORS(cfg.CORSOrigins)(Recover(Logger(mux)))
}
