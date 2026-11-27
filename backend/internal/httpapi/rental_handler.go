package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"unesarent/backend/internal/rental"
)

type RentalHandler struct {
	svc            *rental.Service
	hub            *chatHub
	tickets        *ticketStore
	agreementPath  string
	originPatterns []string
	passwords      passwordReader
	audit          auditLogger
	trustedProxy   bool
}

func NewRentalHandler(svc *rental.Service, agreementPath string, allowedOrigins []string) *RentalHandler {
	return &RentalHandler{
		svc:            svc,
		hub:            newChatHub(),
		tickets:        newTicketStore(),
		agreementPath:  agreementPath,
		originPatterns: allowedOrigins,
	}
}

func (h *RentalHandler) WithAccessPolicy(passwords passwordReader, audit auditLogger, trustedProxy bool) *RentalHandler {
	h.passwords = passwords
	h.audit = audit
	h.trustedProxy = trustedProxy
	return h
}

func (h *RentalHandler) Availability(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	list, err := h.svc.Availability(r.Context(), r.PathValue("id"), q.Get("from"), q.Get("to"))
	if err != nil {
		h.handleError(w, "listing_availability", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"unavailable": list})
}

func (h *RentalHandler) Submit(w http.ResponseWriter, r *http.Request) {
	var in rental.DateInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan field start_date dan end_date")
		return
	}

	rentalItem, err := h.svc.Submit(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), in)
	if err != nil {
		h.handleError(w, "submit_rental", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"rental": rentalItem})
}

func (h *RentalHandler) Mine(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.Mine(
		r.Context(),
		userIDFromContext(r.Context()),
		r.URL.Query().Get("status"),
		queryInt(r, "limit", 0),
		queryInt(r, "offset", 0),
	)
	if err != nil {
		h.handleError(w, "my_rentals", err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *RentalHandler) Detail(w http.ResponseWriter, r *http.Request) {
	rentalItem, err := h.svc.Get(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "rental_detail", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rental": rentalItem})
}

func (h *RentalHandler) Approve(w http.ResponseWriter, r *http.Request) {
	rentalItem, err := h.svc.Approve(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "approve_rental", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rental": rentalItem})
}

func (h *RentalHandler) Reject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan field reason")
		return
	}

	rentalItem, err := h.svc.Reject(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), req.Reason)
	if err != nil {
		h.handleError(w, "reject_rental", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rental": rentalItem})
}

func (h *RentalHandler) Cancel(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan field reason")
		return
	}

	rentalItem, err := h.svc.Cancel(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), req.Reason)
	if err != nil {
		h.handleError(w, "cancel_rental", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rental": rentalItem})
}

func (h *RentalHandler) Complete(w http.ResponseWriter, r *http.Request) {
	rentalItem, err := h.svc.Complete(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "complete_rental", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rental": rentalItem})
}

func (h *RentalHandler) Notifications(w http.ResponseWriter, r *http.Request) {
	unread := r.URL.Query().Get("unread") == "1"
	page, err := h.svc.Notifications(
		r.Context(),
		userIDFromContext(r.Context()),
		unread,
		queryInt(r, "limit", 0),
		queryInt(r, "offset", 0),
	)
	if err != nil {
		h.handleError(w, "list_notifications", err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *RentalHandler) MarkNotificationRead(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.MarkNotificationRead(r.Context(), userIDFromContext(r.Context()), r.PathValue("id")); err != nil {
		h.handleError(w, "read_notification", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *RentalHandler) Messages(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Messages(
		r.Context(),
		userIDFromContext(r.Context()),
		r.PathValue("id"),
		r.URL.Query().Get("after"),
		queryInt(r, "limit", 0),
	)
	if err != nil {
		h.handleError(w, "list_messages", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": list})
}

func (h *RentalHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	var in rental.MessageInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan field body")
		return
	}

	msg, err := h.svc.SendMessage(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), in.Body)
	if err != nil {
		h.handleError(w, "send_message", err)
		return
	}
	h.broadcastMessage(r.Context(), msg)
	writeJSON(w, http.StatusCreated, map[string]any{"message": msg})
}

func (h *RentalHandler) WSTicket(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	rentalID := r.PathValue("id")
	if err := h.svc.ChatAccess(r.Context(), userID, rentalID); err != nil {
		h.handleError(w, "ws_ticket", err)
		return
	}

	token, err := h.tickets.issue(userID, rentalID)
	if err != nil {
		slog.Error("gagal membuat tiket WebSocket", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan pada server")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"ticket":     token,
		"expires_in": int(wsTicketTTL.Seconds()),
		"ws_path":    "/api/rentals/" + rentalID + "/ws?ticket=" + token,
	})
}

func (h *RentalHandler) WebSocket(w http.ResponseWriter, r *http.Request) {
	rentalID := r.PathValue("id")
	ticket, ok := h.tickets.consume(r.URL.Query().Get("ticket"))
	if !ok || ticket.RentalID != rentalID {
		writeError(w, http.StatusUnauthorized, "invalid_ticket", "Tiket WebSocket tidak valid atau sudah kedaluwarsa")
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: h.originPatterns,
	})
	if err != nil {
		return
	}
	defer conn.CloseNow()
	conn.SetReadLimit(4096)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	if err := h.svc.ChatAccess(ctx, ticket.UserID, rentalID); err != nil {
		_ = conn.Close(websocket.StatusCode(4003), "bukan peserta rental")
		return
	}

	client := &chatClient{conn: conn, userID: ticket.UserID}
	h.hub.add(rentalID, client)
	defer h.hub.remove(rentalID, client)

	go h.ping(ctx, client)

	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText {
			h.sendWSError(ctx, client, "invalid_frame", "Hanya frame teks JSON yang didukung")
			continue
		}

		var in struct {
			Type string `json:"type"`
			Body string `json:"body"`
		}
		if err := json.Unmarshal(data, &in); err != nil || in.Type != "message" {
			h.sendWSError(ctx, client, "invalid_frame", "Frame harus {\"type\":\"message\",\"body\":\"...\"}")
			continue
		}
		if !client.allowMessage(time.Now()) {
			h.sendWSError(ctx, client, "rate_limited", "Terlalu banyak pesan, coba lagi nanti")
			continue
		}

		msg, err := h.svc.SendMessage(ctx, ticket.UserID, rentalID, in.Body)
		if err != nil {
			var validation *rental.ValidationError
			switch {
			case errors.As(err, &validation):
				h.sendWSError(ctx, client, "validation_error", "Pesan tidak valid")
			case errors.Is(err, rental.ErrChatClosed):
				h.sendWSError(ctx, client, "chat_closed", "Chat sudah ditutup untuk rental ini")
			case errors.Is(err, rental.ErrForbidden), errors.Is(err, rental.ErrNotFound):
				_ = conn.Close(websocket.StatusCode(4003), "bukan peserta rental")
				return
			default:
				slog.Error("gagal menyimpan pesan chat", "rental", rentalID, "error", err)
				h.sendWSError(ctx, client, "internal_error", "Pesan gagal disimpan")
			}
			continue
		}

		h.broadcastMessage(ctx, msg)
	}
}

func (h *RentalHandler) AgreementFile(w http.ResponseWriter, r *http.Request) {
	_, data, err := h.svc.AgreementFile(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "rental_agreement", err)
		return
	}
	writePDF(w, "surat-perjanjian-sewa.pdf", data)
}

func (h *RentalHandler) ping(ctx context.Context, client *chatClient) {
	ticker := time.NewTicker(wsPingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := client.ping(ctx); err != nil {
				_ = client.conn.CloseNow()
				return
			}
		}
	}
}

type wsMessageFrame struct {
	Type    string         `json:"type"`
	Message rental.Message `json:"message"`
}

type wsErrorFrame struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *RentalHandler) broadcastMessage(ctx context.Context, msg rental.Message) {
	data, err := json.Marshal(wsMessageFrame{Type: "message", Message: msg})
	if err != nil {
		slog.Error("gagal menyusun frame chat", "error", err)
		return
	}
	h.hub.broadcast(ctx, msg.RentalID, data)
}

func (h *RentalHandler) sendWSError(ctx context.Context, client *chatClient, code, message string) {
	data, err := json.Marshal(wsErrorFrame{Type: "error", Code: code, Message: message})
	if err != nil {
		return
	}
	_ = client.write(ctx, data)
}

func (h *RentalHandler) handleError(w http.ResponseWriter, operation string, err error) {
	var validation *rental.ValidationError
	if errors.As(err, &validation) {
		writeValidationError(w, validation.Fields)
		return
	}

	switch {
	case errors.Is(err, rental.ErrNotFound):
		writeError(w, http.StatusNotFound, "rental_not_found", "Rental tidak ditemukan")

	case errors.Is(err, rental.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Anda tidak memiliki akses ke rental ini")

	case errors.Is(err, rental.ErrNotVerified):
		writeError(w, http.StatusForbidden, "ktm_not_verified", "Akun harus terverifikasi KTM untuk menyewa")

	case errors.Is(err, rental.ErrInvalidTransition):
		writeError(w, http.StatusConflict, "rental_invalid_transition", "Status rental tidak memungkinkan aksi ini")

	case errors.Is(err, rental.ErrInvalidDates):
		writeValidationError(w, map[string]string{
			"start_date": "Tanggal rental tidak valid",
			"end_date":   "Tanggal rental tidak valid",
		})

	case errors.Is(err, rental.ErrReasonRequired):
		writeValidationError(w, map[string]string{"reason": "Alasan wajib diisi"})

	case errors.Is(err, rental.ErrDateUnavailable):
		writeError(w, http.StatusConflict, "date_unavailable", "Tanggal sudah dipesan rental lain")

	case errors.Is(err, rental.ErrDatesBlocked):
		writeError(w, http.StatusConflict, "dates_blocked", "Tanggal diblokir penyedia")

	case errors.Is(err, rental.ErrChatClosed):
		writeError(w, http.StatusConflict, "chat_closed", "Chat sudah ditutup untuk rental ini")

	case errors.Is(err, rental.ErrInvalidBody):
		writeValidationError(w, map[string]string{"body": "Pesan tidak valid"})

	case errors.Is(err, rental.ErrCODIncomplete):
		writeError(w, http.StatusConflict, "cod_incomplete", "Selesaikan serah terima dan pengembalian barang sebelum menandai selesai.")

	case errors.Is(err, rental.ErrInvalidCODState):
		writeError(w, http.StatusConflict, "cod_invalid_state", "State COD tidak memungkinkan aksi ini")

	case errors.Is(err, rental.ErrMissingProof):
		writeValidationError(w, map[string]string{"file": "Bukti wajib diunggah"})

	case errors.Is(err, rental.ErrDocumentNotFound):
		writeError(w, http.StatusNotFound, "document_not_found", "Dokumen tidak ditemukan")

	case errors.Is(err, rental.ErrInvalidReason):
		writeValidationError(w, map[string]string{"reason": "Alasan harus 10-1000 karakter"})

	case errors.Is(err, rental.ErrInvalidPassword):
		writeError(w, http.StatusUnauthorized, "invalid_password", "Kata sandi admin tidak benar")

	case errors.Is(err, rental.ErrAgreementMissing):
		writeError(w, http.StatusNotFound, "agreement_not_found", "Surat perjanjian tidak tersedia")

	default:
		slog.Error("kesalahan rental", "operation", operation, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan pada server")
	}
}
