package httpapi

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"

	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/rental"
	"unesarent/backend/internal/user"
)

type passwordReader interface {
	FindByID(ctx context.Context, id string) (user.User, error)
}

type auditLogger interface {
	LogAction(ctx context.Context, actorID, action, targetType, targetID, reason string, metadata map[string]any) error
}

const maxProofBytes = 10 << 20

func rentalFilterFromRequest(r *http.Request) rental.RentalFilter {
	q := r.URL.Query()
	overdue, _ := strconv.ParseBool(strings.TrimSpace(q.Get("overdue")))
	return rental.RentalFilter{
		Role:    q.Get("role"),
		Status:  q.Get("status"),
		From:    q.Get("from"),
		To:      q.Get("to"),
		Query:   q.Get("q"),
		Overdue: overdue,
		Limit:   queryInt(r, "limit", 20),
		Offset:  queryInt(r, "offset", 0),
	}
}

func (h *RentalHandler) MineFiltered(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.MineFiltered(r.Context(), userIDFromContext(r.Context()), rentalFilterFromRequest(r))
	if err != nil {
		h.handleError(w, "rental_mine", err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *RentalHandler) History(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.History(r.Context(), userIDFromContext(r.Context()),
		r.URL.Query().Get("role"), r.URL.Query().Get("year"))
	if err != nil {
		h.handleError(w, "rental_history", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"history": summary})
}

func (h *RentalHandler) Documents(w http.ResponseWriter, r *http.Request) {
	docs, err := h.svc.Documents(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "rental_documents", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": docs})
}

func (h *RentalHandler) DocumentFile(w http.ResponseWriter, r *http.Request) {
	doc, data, err := h.svc.DocumentFile(r.Context(), userIDFromContext(r.Context()),
		r.PathValue("id"), r.PathValue("docID"))
	if err != nil {
		h.handleError(w, "rental_document_file", err)
		return
	}
	writeAttachment(w, documentFilename(doc.Type), doc.MimeType, data)
}

func documentFilename(docType string) string {
	switch docType {
	case "agreement":
		return "surat-perjanjian-sewa.pdf"
	case "cod_receipt":
		return "kuitansi-cod.pdf"
	case "handover_proof":
		return "bukti-serah-terima.jpg"
	case "return_proof":
		return "bukti-pengembalian.jpg"
	default:
		return "dokumen-rental.pdf"
	}
}

func readProof(w http.ResponseWriter, r *http.Request) ([]byte, string, string, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxProofBytes+(1<<20))
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Form bukti tidak valid atau terlalu besar")
		return nil, "", "", false
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "Bukti foto wajib diunggah")
		return nil, "", "", false
	}
	defer file.Close()
	mimeType := header.Header.Get("Content-Type")
	if mimeType != "image/jpeg" && mimeType != "image/png" {
		writeValidationError(w, map[string]string{"file": "Bukti harus JPG atau PNG"})
		return nil, "", "", false
	}
	data, err := io.ReadAll(io.LimitReader(file, maxProofBytes+1))
	if err != nil || len(data) > maxProofBytes {
		writeError(w, http.StatusBadRequest, "file_too_large", "Ukuran bukti maksimal 10 MB")
		return nil, "", "", false
	}
	if len(data) == 0 {
		writeError(w, http.StatusBadRequest, "missing_file", "Bukti foto wajib diunggah")
		return nil, "", "", false
	}
	return data, mimeType, r.FormValue("note"), true
}

func (h *RentalHandler) COD(w http.ResponseWriter, r *http.Request) {
	state, err := h.svc.COD(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "rental_cod", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cod": state})
}

func (h *RentalHandler) Handover(w http.ResponseWriter, r *http.Request) {
	proof, mime, note, ok := readProof(w, r)
	if !ok {
		return
	}
	doc, err := h.svc.Handover(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), note, proof, mime)
	if err != nil {
		h.handleError(w, "rental_handover", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": doc})
}

func (h *RentalHandler) ConfirmHandover(w http.ResponseWriter, r *http.Request) {
	state, err := h.svc.ConfirmHandover(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "rental_handover_confirm", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cod": state})
}

func (h *RentalHandler) ReturnProof(w http.ResponseWriter, r *http.Request) {
	proof, mime, note, ok := readProof(w, r)
	if !ok {
		return
	}
	doc, err := h.svc.Return(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), note, proof, mime)
	if err != nil {
		h.handleError(w, "rental_return", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": doc})
}

func (h *RentalHandler) ConfirmReturn(w http.ResponseWriter, r *http.Request) {
	state, err := h.svc.ConfirmReturn(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "rental_return_confirm", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"cod": state})
}

func (h *RentalHandler) Settle(w http.ResponseWriter, r *http.Request) {
	var in rental.CODSettleInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Isi permintaan tidak valid")
		return
	}
	doc, err := h.svc.Settle(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), in)
	if err != nil {
		h.handleError(w, "rental_settle", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"document": doc})
}

func (h *RentalHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.AdminList(r.Context(), rentalFilterFromRequest(r))
	if err != nil {
		h.handleError(w, "admin_rentals", err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *RentalHandler) AdminDetail(w http.ResponseWriter, r *http.Request) {
	item, err := h.svc.AdminGet(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "admin_rental_detail", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rental": item})
}

func (h *RentalHandler) AdminStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.svc.AdminStats(r.Context())
	if err != nil {
		h.handleError(w, "admin_rental_stats", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"stats": stats})
}

func (h *RentalHandler) ExportRentals(w http.ResponseWriter, r *http.Request) {
	items, err := h.svc.ExportRentals(r.Context(), rentalFilterFromRequest(r))
	if err != nil {
		h.handleError(w, "admin_rental_export", err)
		return
	}
	rows := make([][]string, 0, len(items))
	for _, item := range items {
		rows = append(rows, []string{
			item.ID, item.Listing.Title, item.Renter.Name, item.Owner.Name,
			item.StartDate, item.EndDate, csvInt(item.TotalDays), csvInt(item.TotalPrice),
			string(item.Status), item.CreatedAt.Format("2006-01-02 15:04"),
		})
	}
	ip := clientIP(r, h.trustedProxy)
	if h.audit != nil {
		_ = h.audit.LogAction(r.Context(), userIDFromContext(r.Context()), "export_data", "rental",
			userIDFromContext(r.Context()), "Ekspor data rental oleh admin",
			map[string]any{"ip": ip, "user_agent": r.UserAgent()})
	}
	writeCSV(w, "penyewaan.csv",
		[]string{"ID", "Barang", "Penyewa", "Penyedia", "Mulai", "Selesai", "Hari", "Total", "Status", "Dibuat"}, rows)
}

func (h *RentalHandler) AdminDocuments(w http.ResponseWriter, r *http.Request) {
	docs, err := h.svc.AdminDocuments(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "admin_documents", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": docs})
}

func (h *RentalHandler) AdminDocumentDownload(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Isi permintaan tidak valid")
		return
	}
	if n := len(strings.TrimSpace(body.Reason)); n < 10 || n > 1000 {
		writeValidationError(w, map[string]string{"reason": "Alasan harus 10-1000 karakter"})
		return
	}
	doc, data, err := h.svc.AdminDocumentFile(r.Context(), r.PathValue("id"), r.PathValue("docID"))
	if err != nil {
		h.handleError(w, "admin_document_download", err)
		return
	}
	ip := clientIP(r, h.trustedProxy)
	if h.audit != nil {
		_ = h.audit.LogAction(r.Context(), userIDFromContext(r.Context()), "download_document", "rental_document",
			doc.ID, strings.TrimSpace(body.Reason), map[string]any{"ip": ip, "user_agent": r.UserAgent()})
	}
	writeAttachment(w, documentFilename(doc.Type), doc.MimeType, data)
}

func (h *RentalHandler) AdminMessages(w http.ResponseWriter, r *http.Request) {
	items, total, err := h.svc.AdminMessageMetadata(r.Context(), r.PathValue("id"),
		queryInt(r, "limit", 50), queryInt(r, "offset", 0))
	if err != nil {
		h.handleError(w, "admin_messages", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"messages": items, "total": total,
		"limit": queryInt(r, "limit", 50), "offset": queryInt(r, "offset", 0),
	})
}

type adminSensitiveBody struct {
	Reason   string `json:"reason"`
	Password string `json:"password"`
}

func (h *RentalHandler) verifyAdminPassword(w http.ResponseWriter, r *http.Request, body adminSensitiveBody) bool {
	if h.passwords == nil {
		writeError(w, http.StatusForbidden, "forbidden", "Verifikasi kata sandi tidak tersedia")
		return false
	}
	adminID := userIDFromContext(r.Context())
	record, err := h.passwords.FindByID(r.Context(), adminID)
	if err != nil || !auth.CheckPassword(record.PasswordHash, body.Password) {
		writeError(w, http.StatusUnauthorized, "invalid_password", "Kata sandi admin tidak benar")
		return false
	}
	return true
}

func (h *RentalHandler) AdminReveal(w http.ResponseWriter, r *http.Request) {
	var body adminSensitiveBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Isi permintaan tidak valid")
		return
	}
	if !h.verifyAdminPassword(w, r, body) {
		return
	}
	ip, ua := clientIP(r, h.trustedProxy), r.UserAgent()
	plain, err := h.svc.AdminRevealMessage(r.Context(), userIDFromContext(r.Context()),
		r.PathValue("id"), r.PathValue("msgID"), body.Reason, ip, ua)
	if err != nil {
		h.handleError(w, "admin_reveal", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"body": plain, "audit": true})
}

func (h *RentalHandler) AdminExportTranscript(w http.ResponseWriter, r *http.Request) {
	var body adminSensitiveBody
	if err := decodeJSON(w, r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Isi permintaan tidak valid")
		return
	}
	if !h.verifyAdminPassword(w, r, body) {
		return
	}
	ip, ua := clientIP(r, h.trustedProxy), r.UserAgent()
	data, err := h.svc.AdminExportTranscript(r.Context(), userIDFromContext(r.Context()),
		r.PathValue("id"), body.Reason, ip, ua)
	if err != nil {
		h.handleError(w, "admin_transcript", err)
		return
	}
	writeAttachment(w, "transkrip-chat.txt", "text/plain; charset=utf-8", data)
}
