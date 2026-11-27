package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"unesarent/backend/internal/admin"
)

type AdminHandler struct {
	svc          *admin.Service
	trustedProxy bool
}

func NewAdminHandler(svc *admin.Service, trustedProxy bool) *AdminHandler {
	return &AdminHandler{svc: svc, trustedProxy: trustedProxy}
}

type reasonBody struct {
	Reason string `json:"reason"`
}

func (h *AdminHandler) requestInfo(r *http.Request) (string, string) {
	return clientIP(r, h.trustedProxy), r.UserAgent()
}

func (h *AdminHandler) handleError(w http.ResponseWriter, err error) {
	var validation *admin.ValidationError
	switch {
	case errors.As(err, &validation):
		writeValidationError(w, validation.Fields)
	case errors.Is(err, admin.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Data tidak ditemukan")
	case errors.Is(err, admin.ErrUserHasActivity):
		writeError(w, http.StatusConflict, "user_has_activity", "Pengguna memiliki aktivitas, tidak bisa dihapus")
	case errors.Is(err, admin.ErrUserVerified):
		writeError(w, http.StatusConflict, "user_verified", "Hanya akun belum terverifikasi yang bisa dihapus")
	case errors.Is(err, admin.ErrCannotSuspendSelf), errors.Is(err, admin.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Tindakan tidak diizinkan")
	default:
		slog.Error("admin handler", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan pada server")
	}
}

// --- Laporan peserta ---

func (h *AdminHandler) CreateReport(w http.ResponseWriter, r *http.Request) {
	var in admin.ReportInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Isi permintaan tidak valid")
		return
	}
	report, err := h.svc.CreateReport(r.Context(), userIDFromContext(r.Context()), in)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"report": report})
}

func (h *AdminHandler) MyReports(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.MyReports(r.Context(), userIDFromContext(r.Context()),
		queryInt(r, "limit", 20), queryInt(r, "offset", 0))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

// --- Antrean laporan admin ---

func (h *AdminHandler) Reports(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.Reports(r.Context(), admin.ReportFilter{
		Status: r.URL.Query().Get("status"),
		Type:   r.URL.Query().Get("type"),
		Limit:  queryInt(r, "limit", 20),
		Offset: queryInt(r, "offset", 0),
	})
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *AdminHandler) ReportDetail(w http.ResponseWriter, r *http.Request) {
	report, err := h.svc.Report(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"report": report})
}

func (h *AdminHandler) moderateReport(w http.ResponseWriter, r *http.Request, action string) {
	var body struct {
		ResolutionNote string `json:"resolution_note"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Isi permintaan tidak valid")
		return
	}
	ip, ua := h.requestInfo(r)
	report, err := h.svc.ResolveReport(r.Context(), userIDFromContext(r.Context()),
		r.PathValue("id"), action, body.ResolutionNote, ip, ua)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"report": report})
}

func (h *AdminHandler) ResolveReport(w http.ResponseWriter, r *http.Request) {
	h.moderateReport(w, r, "resolve")
}

func (h *AdminHandler) RejectReport(w http.ResponseWriter, r *http.Request) {
	h.moderateReport(w, r, "reject")
}

// --- Pengguna ---

func (h *AdminHandler) Users(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.Users(r.Context(), userFilterFromRequest(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func userFilterFromRequest(r *http.Request) admin.UserFilter {
	return admin.UserFilter{
		Query:    r.URL.Query().Get("q"),
		Role:     r.URL.Query().Get("role"),
		Status:   r.URL.Query().Get("status"),
		Verified: r.URL.Query().Get("verified"),
	}
}

func (h *AdminHandler) UserDetail(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.User(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": item})
}

func (h *AdminHandler) ExportUsers(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ExportUsers(r.Context(), userFilterFromRequest(r))
	if err != nil {
		h.handleError(w, err)
		return
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID, item.Name, item.Email, item.Role, item.Status,
			item.NIM, strconv.FormatBool(item.IsVerified),
			item.CreatedAt.Format("2006-01-02 15:04"), csvTime(item.LastLoginAt),
		})
	}
	ip, ua := h.requestInfo(r)
	h.svc.LogExport(r.Context(), userIDFromContext(r.Context()), "user",
		"Ekspor data pengguna oleh admin", ip, ua)
	writeCSV(w, "pengguna.csv",
		[]string{"ID", "Nama", "Email", "Peran", "Status", "NIM", "Terverifikasi", "Dibuat", "Login Terakhir"}, rows)
}

func (h *AdminHandler) moderateUser(w http.ResponseWriter, r *http.Request, action string) {
	var body reasonBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Isi permintaan tidak valid")
		return
	}
	ip, ua := h.requestInfo(r)
	adminID, id := userIDFromContext(r.Context()), r.PathValue("id")
	var err error
	switch action {
	case "suspend":
		err = h.svc.Suspend(r.Context(), adminID, id, body.Reason, ip, ua)
	case "reactivate":
		err = h.svc.Reactivate(r.Context(), adminID, id, body.Reason, ip, ua)
	default:
		err = h.svc.DeleteUser(r.Context(), adminID, id, body.Reason, ip, ua)
	}
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (h *AdminHandler) SuspendUser(w http.ResponseWriter, r *http.Request) {
	h.moderateUser(w, r, "suspend")
}

func (h *AdminHandler) ReactivateUser(w http.ResponseWriter, r *http.Request) {
	h.moderateUser(w, r, "reactivate")
}

func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	h.moderateUser(w, r, "delete")
}

// --- Audit & statistik ---

func (h *AdminHandler) Audit(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.Audit(r.Context(), admin.AuditFilter{
		ActorID: r.URL.Query().Get("actor_id"),
		Action:  r.URL.Query().Get("action"),
		From:    r.URL.Query().Get("from"),
		To:      r.URL.Query().Get("to"),
		Limit:   queryInt(r, "limit", 20),
		Offset:  queryInt(r, "offset", 0),
	})
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *AdminHandler) Stats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.Stats(r.Context())
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}

// parseCSVDate dipakai oleh handler ekspor rental.
func parseCSVDate(raw string) string {
	t, err := time.Parse("2006-01-02", raw)
	if err != nil {
		return raw
	}
	return t.Format("2006-01-02")
}
