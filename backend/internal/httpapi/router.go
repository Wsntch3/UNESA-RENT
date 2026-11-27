package httpapi

import (
	"net/http"
	"time"

	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/config"
	"unesarent/backend/internal/user"
)

func NewRouter(cfg config.Config, authHandler *AuthHandler, ktmHandler *KTMHandler, listingHandler *ListingHandler, rentalHandler *RentalHandler, adminHandler *AdminHandler, tokens *auth.TokenIssuer, states AuthStateReader) http.Handler {
	mux := http.NewServeMux()
	limit := RateLimit(10, time.Minute, cfg.TrustedProxy)
	registerLimit := RateLimit(5, 10*time.Minute, cfg.TrustedProxy)
	otpLimit := RateLimit(10, time.Minute, cfg.TrustedProxy)

	mux.HandleFunc("GET /api/health", Health)
	mux.HandleFunc("GET /api/health/live", HealthLive)
	mux.HandleFunc("GET /api/health/ready", HealthReady)
	mux.Handle("POST /api/auth/register", registerLimit(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /api/auth/login", limit(http.HandlerFunc(authHandler.Login)))
	mux.Handle("POST /api/auth/otp/request", otpLimit(http.HandlerFunc(authHandler.RequestOTP)))
	mux.Handle("POST /api/auth/otp/verify", otpLimit(http.HandlerFunc(authHandler.VerifyOTP)))
	mux.Handle("POST /api/auth/login/otp", otpLimit(http.HandlerFunc(authHandler.LoginOTP)))
	mux.Handle("POST /api/auth/password/forgot", otpLimit(http.HandlerFunc(authHandler.ForgotPassword)))
	mux.Handle("POST /api/auth/password/reset", otpLimit(http.HandlerFunc(authHandler.ResetPassword)))
	mux.Handle("GET /api/auth/me", RequireAuth(tokens, states)(http.HandlerFunc(authHandler.Me)))

	if ktmHandler != nil {
		requireAuth := RequireAuth(tokens, states)
		requireMahasiswa := RequireRole(string(user.RoleMahasiswa))
		requireAdmin := RequireRole(string(user.RoleAdminDev))
		ktmLimit := RateLimit(5, time.Hour, cfg.TrustedProxy)

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
		photoLimit := RateLimit(30, time.Minute, cfg.TrustedProxy)

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
		messageLimit := RateLimit(60, time.Minute, cfg.TrustedProxy)
		ticketLimit := RateLimit(10, time.Minute, cfg.TrustedProxy)

		mux.HandleFunc("GET /api/listings/{id}/availability", rentalHandler.Availability)

		mux.Handle("POST /api/listings/{id}/rentals", requireAuth(requireMahasiswa(http.HandlerFunc(rentalHandler.Submit))))
		mux.Handle("GET /api/rentals/mine", requireAuth(http.HandlerFunc(rentalHandler.MineFiltered)))
		mux.Handle("GET /api/rentals/history", requireAuth(http.HandlerFunc(rentalHandler.History)))
		mux.Handle("GET /api/rentals/{id}", requireAuth(http.HandlerFunc(rentalHandler.Detail)))
		mux.Handle("POST /api/rentals/{id}/approve", requireAuth(http.HandlerFunc(rentalHandler.Approve)))
		mux.Handle("POST /api/rentals/{id}/reject", requireAuth(http.HandlerFunc(rentalHandler.Reject)))
		mux.Handle("POST /api/rentals/{id}/cancel", requireAuth(http.HandlerFunc(rentalHandler.Cancel)))
		mux.Handle("POST /api/rentals/{id}/complete", requireAuth(http.HandlerFunc(rentalHandler.Complete)))
		mux.Handle("GET /api/rentals/{id}/agreement/file", requireAuth(http.HandlerFunc(rentalHandler.AgreementFile)))

		mux.Handle("GET /api/rentals/{id}/documents", requireAuth(http.HandlerFunc(rentalHandler.Documents)))
		mux.Handle("GET /api/rentals/{id}/documents/{docID}/file", requireAuth(http.HandlerFunc(rentalHandler.DocumentFile)))
		mux.Handle("GET /api/rentals/{id}/cod", requireAuth(http.HandlerFunc(rentalHandler.COD)))
		mux.Handle("POST /api/rentals/{id}/cod/handover", requireAuth(http.HandlerFunc(rentalHandler.Handover)))
		mux.Handle("POST /api/rentals/{id}/cod/handover/confirm", requireAuth(http.HandlerFunc(rentalHandler.ConfirmHandover)))
		mux.Handle("POST /api/rentals/{id}/cod/return", requireAuth(http.HandlerFunc(rentalHandler.ReturnProof)))
		mux.Handle("POST /api/rentals/{id}/cod/return/confirm", requireAuth(http.HandlerFunc(rentalHandler.ConfirmReturn)))
		mux.Handle("POST /api/rentals/{id}/cod/settle", requireAuth(http.HandlerFunc(rentalHandler.Settle)))

		mux.Handle("GET /api/rentals/{id}/messages", requireAuth(http.HandlerFunc(rentalHandler.Messages)))
		mux.Handle("POST /api/rentals/{id}/messages", messageLimit(requireAuth(http.HandlerFunc(rentalHandler.SendMessage))))
		mux.Handle("POST /api/rentals/{id}/ws-ticket", ticketLimit(requireAuth(http.HandlerFunc(rentalHandler.WSTicket))))
		// Handshake WebSocket memakai tiket sekali pakai, bukan header Authorization.
		mux.HandleFunc("GET /api/rentals/{id}/ws", rentalHandler.WebSocket)

		mux.Handle("GET /api/notifications", requireAuth(http.HandlerFunc(rentalHandler.Notifications)))
		mux.Handle("POST /api/notifications/{id}/read", requireAuth(http.HandlerFunc(rentalHandler.MarkNotificationRead)))

		requireAdmin := RequireRole(string(user.RoleAdminDev))
		revealLimit := RateLimit(10, time.Hour, cfg.TrustedProxy)
		mux.Handle("GET /api/admin/rentals", requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.AdminList))))
		mux.Handle("GET /api/admin/rentals/stats", requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.AdminStats))))
		mux.Handle("GET /api/admin/rentals/export.csv", requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.ExportRentals))))
		mux.Handle("GET /api/admin/rentals/{id}", requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.AdminDetail))))
		mux.Handle("GET /api/admin/rentals/{id}/documents", requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.AdminDocuments))))
		mux.Handle("POST /api/admin/rentals/{id}/documents/{docID}/download", requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.AdminDocumentDownload))))
		mux.Handle("GET /api/admin/rentals/{id}/messages", requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.AdminMessages))))
		mux.Handle("POST /api/admin/rentals/{id}/messages/{msgID}/reveal", revealLimit(requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.AdminReveal)))))
		mux.Handle("POST /api/admin/rentals/{id}/transcript/export", revealLimit(requireAuth(requireAdmin(http.HandlerFunc(rentalHandler.AdminExportTranscript)))))
	}

	if adminHandler != nil {
		requireAuth := RequireAuth(tokens, states)
		requireAdmin := RequireRole(string(user.RoleAdminDev))
		requireMahasiswa := RequireRole(string(user.RoleMahasiswa))

		mux.Handle("POST /api/reports", requireAuth(requireMahasiswa(http.HandlerFunc(adminHandler.CreateReport))))
		mux.Handle("GET /api/reports/mine", requireAuth(http.HandlerFunc(adminHandler.MyReports)))

		mux.Handle("GET /api/admin/reports", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.Reports))))
		mux.Handle("GET /api/admin/reports/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.ReportDetail))))
		mux.Handle("POST /api/admin/reports/{id}/resolve", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.ResolveReport))))
		mux.Handle("POST /api/admin/reports/{id}/reject", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.RejectReport))))

		mux.Handle("GET /api/admin/users", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.Users))))
		mux.Handle("GET /api/admin/users/export.csv", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.ExportUsers))))
		mux.Handle("GET /api/admin/users/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.UserDetail))))
		mux.Handle("POST /api/admin/users/{id}/suspend", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.SuspendUser))))
		mux.Handle("POST /api/admin/users/{id}/reactivate", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.ReactivateUser))))
		mux.Handle("DELETE /api/admin/users/{id}", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.DeleteUser))))

		mux.Handle("GET /api/admin/audit", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.Audit))))
		mux.Handle("GET /api/admin/stats", requireAuth(requireAdmin(http.HandlerFunc(adminHandler.Stats))))
	}

	return CORS(cfg.CORSOrigins)(Recover(Logger(SecurityHeaders(mux))))
}
