package httpapi

import (
	"net/http"
	"time"

	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/config"
	"unesarent/backend/internal/user"
)

func NewRouter(cfg config.Config, authHandler *AuthHandler, ktmHandler *KTMHandler, listingHandler *ListingHandler, rentalHandler *RentalHandler, tokens *auth.TokenIssuer, states AuthStateReader) http.Handler {
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

	if listingHandler != nil {
		requireAuth := RequireAuth(tokens, states)
		requireMahasiswa := RequireRole(string(user.RoleMahasiswa))
		requireAdmin := RequireRole(string(user.RoleAdminDev))
		photoLimit := RateLimit(30, time.Minute)

		// Katalog publik.
		mux.HandleFunc("GET /api/categories", listingHandler.Categories)
		mux.HandleFunc("GET /api/campus-locations", listingHandler.Locations)
		mux.HandleFunc("GET /api/listings", listingHandler.Search)
		mux.HandleFunc("GET /api/listings/{id}/photos/{photoID}/file", listingHandler.PhotoFile)
		mux.HandleFunc("GET /api/listings/{id}", listingHandler.Detail)

		// Pengelolaan listing oleh penyedia.
		mux.Handle("GET /api/listings/mine", requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.Mine))))
		mux.Handle("POST /api/listings", requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.Create))))
		mux.Handle("PUT /api/listings/{id}", requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.Update))))
		mux.Handle("DELETE /api/listings/{id}", requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.Delete))))
		mux.Handle("POST /api/listings/{id}/photos", photoLimit(requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.AddPhoto)))))
		mux.Handle("DELETE /api/listings/{id}/photos/{photoID}", requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.DeletePhoto))))
		mux.Handle("GET /api/listings/{id}/blocks", requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.Blocks))))
		mux.Handle("POST /api/listings/{id}/blocks", requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.AddBlock))))
		mux.Handle("DELETE /api/listings/{id}/blocks/{blockID}", requireAuth(requireMahasiswa(http.HandlerFunc(listingHandler.DeleteBlock))))

		// Moderasi admin.
		mux.Handle("GET /api/admin/listings", requireAuth(requireAdmin(http.HandlerFunc(listingHandler.AdminList))))
		mux.Handle("POST /api/admin/listings/{id}/takedown", requireAuth(requireAdmin(http.HandlerFunc(listingHandler.Takedown))))
		mux.Handle("POST /api/admin/listings/{id}/restore", requireAuth(requireAdmin(http.HandlerFunc(listingHandler.Restore))))
	}

	if rentalHandler != nil {
		requireAuth := RequireAuth(tokens, states)
		requireMahasiswa := RequireRole(string(user.RoleMahasiswa))
		messageLimit := RateLimit(60, time.Minute)
		ticketLimit := RateLimit(10, time.Minute)

		mux.HandleFunc("GET /api/listings/{id}/availability", rentalHandler.Availability)

		mux.Handle("POST /api/listings/{id}/rentals", requireAuth(requireMahasiswa(http.HandlerFunc(rentalHandler.Submit))))
		mux.Handle("GET /api/rentals/mine", requireAuth(http.HandlerFunc(rentalHandler.Mine)))
		mux.Handle("GET /api/rentals/{id}", requireAuth(http.HandlerFunc(rentalHandler.Detail)))
		mux.Handle("POST /api/rentals/{id}/approve", requireAuth(http.HandlerFunc(rentalHandler.Approve)))
		mux.Handle("POST /api/rentals/{id}/reject", requireAuth(http.HandlerFunc(rentalHandler.Reject)))
		mux.Handle("POST /api/rentals/{id}/cancel", requireAuth(http.HandlerFunc(rentalHandler.Cancel)))
		mux.Handle("POST /api/rentals/{id}/complete", requireAuth(http.HandlerFunc(rentalHandler.Complete)))
		mux.Handle("GET /api/rentals/{id}/agreement/file", requireAuth(http.HandlerFunc(rentalHandler.AgreementFile)))

		mux.Handle("GET /api/rentals/{id}/messages", requireAuth(http.HandlerFunc(rentalHandler.Messages)))
		mux.Handle("POST /api/rentals/{id}/messages", messageLimit(requireAuth(http.HandlerFunc(rentalHandler.SendMessage))))
		mux.Handle("POST /api/rentals/{id}/ws-ticket", ticketLimit(requireAuth(http.HandlerFunc(rentalHandler.WSTicket))))
		// Handshake WebSocket memakai tiket sekali pakai, bukan header Authorization.
		mux.HandleFunc("GET /api/rentals/{id}/ws", rentalHandler.WebSocket)

		mux.Handle("GET /api/notifications", requireAuth(http.HandlerFunc(rentalHandler.Notifications)))
		mux.Handle("POST /api/notifications/{id}/read", requireAuth(http.HandlerFunc(rentalHandler.MarkNotificationRead)))
	}

	return CORS(cfg.CORSOrigins)(Recover(Logger(mux)))
}
