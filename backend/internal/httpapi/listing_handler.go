package httpapi

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"unesarent/backend/internal/catalog"
)

type ListingHandler struct {
	svc *catalog.Service
}

func NewListingHandler(svc *catalog.Service) *ListingHandler {
	return &ListingHandler{svc: svc}
}

func (h *ListingHandler) Categories(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Categories(r.Context())
	if err != nil {
		h.handleError(w, "list_categories", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": list})
}

func (h *ListingHandler) Locations(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Locations(r.Context())
	if err != nil {
		h.handleError(w, "list_locations", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"campus_locations": list})
}

func (h *ListingHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, err := h.svc.Search(r.Context(), catalog.SearchParams{
		Query:    q.Get("q"),
		Category: q.Get("category"),
		Location: q.Get("location"),
		MinPrice: q.Get("min_price"),
		MaxPrice: q.Get("max_price"),
		Sort:     q.Get("sort"),
		Limit:    queryInt(r, "limit", 0),
		Offset:   queryInt(r, "offset", 0),
	})
	if err != nil {
		h.handleError(w, "search_listings", err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *ListingHandler) Detail(w http.ResponseWriter, r *http.Request) {
	listing, err := h.svc.Detail(r.Context(), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "listing_detail", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"listing": listing})
}

func (h *ListingHandler) Mine(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.Mine(
		r.Context(),
		userIDFromContext(r.Context()),
		queryInt(r, "limit", 0),
		queryInt(r, "offset", 0),
	)
	if err != nil {
		h.handleError(w, "my_listings", err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *ListingHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in catalog.ListingInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body request harus berupa JSON yang valid")
		return
	}

	listing, err := h.svc.Create(r.Context(), userIDFromContext(r.Context()), in)
	if err != nil {
		h.handleError(w, "create_listing", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"listing": listing})
}

func (h *ListingHandler) Update(w http.ResponseWriter, r *http.Request) {
	var in catalog.ListingInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body request harus berupa JSON yang valid")
		return
	}

	listing, err := h.svc.Update(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), in)
	if err != nil {
		h.handleError(w, "update_listing", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"listing": listing})
}

func (h *ListingHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), userIDFromContext(r.Context()), r.PathValue("id")); err != nil {
		h.handleError(w, "delete_listing", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListingHandler) AddPhoto(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, catalog.MaxPhotoBytes+maxMultipartOverhead)

	if err := r.ParseMultipartForm(catalog.MaxPhotoBytes); err != nil {
		if strings.Contains(err.Error(), "request body too large") {
			writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Ukuran file melebihi 5 MB")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_body", "Gunakan multipart/form-data dengan field 'file'")
		return
	}
	defer r.MultipartForm.RemoveAll()

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing_file", "Field 'file' wajib diisi")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, catalog.MaxPhotoBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_file", "Gagal membaca file foto")
		return
	}
	if int64(len(data)) > catalog.MaxPhotoBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Ukuran file melebihi 5 MB")
		return
	}

	photo, err := h.svc.AddPhoto(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), data)
	if err != nil {
		h.handleError(w, "add_listing_photo", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"photo": photo})
}

func (h *ListingHandler) DeletePhoto(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeletePhoto(
		r.Context(),
		userIDFromContext(r.Context()),
		r.PathValue("id"),
		r.PathValue("photoID"),
	)
	if err != nil {
		h.handleError(w, "delete_listing_photo", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListingHandler) PhotoFile(w http.ResponseWriter, r *http.Request) {
	photo, file, err := h.svc.PhotoFile(r.Context(), r.PathValue("id"), r.PathValue("photoID"))
	if err != nil {
		h.handleError(w, "listing_photo_file", err)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", photo.MimeType)
	w.Header().Set("Content-Length", strconv.FormatInt(photo.FileSizeBytes, 10))
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)

	if _, err := io.Copy(w, file); err != nil {
		slog.Warn("gagal mengirim foto listing", "id", photo.ID, "error", err)
	}
}

func (h *ListingHandler) Blocks(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.Blocks(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "list_listing_blocks", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"blocks": list})
}

func (h *ListingHandler) AddBlock(w http.ResponseWriter, r *http.Request) {
	var in catalog.BlockInput
	if err := decodeJSON(w, r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body request harus berupa JSON yang valid")
		return
	}

	block, err := h.svc.AddBlock(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), in)
	if err != nil {
		h.handleError(w, "add_listing_block", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"block": block})
}

func (h *ListingHandler) DeleteBlock(w http.ResponseWriter, r *http.Request) {
	err := h.svc.DeleteBlock(
		r.Context(),
		userIDFromContext(r.Context()),
		r.PathValue("id"),
		r.PathValue("blockID"),
	)
	if err != nil {
		h.handleError(w, "delete_listing_block", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *ListingHandler) AdminList(w http.ResponseWriter, r *http.Request) {
	page, err := h.svc.AdminList(
		r.Context(),
		r.URL.Query().Get("status"),
		r.URL.Query().Get("q"),
		queryInt(r, "limit", 0),
		queryInt(r, "offset", 0),
	)
	if err != nil {
		h.handleError(w, "admin_list_listings", err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func (h *ListingHandler) Takedown(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Body harus JSON dengan field reason")
		return
	}

	listing, err := h.svc.Takedown(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"), req.Reason)
	if err != nil {
		h.handleError(w, "takedown_listing", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"listing": listing})
}

func (h *ListingHandler) Restore(w http.ResponseWriter, r *http.Request) {
	listing, err := h.svc.Restore(r.Context(), userIDFromContext(r.Context()), r.PathValue("id"))
	if err != nil {
		h.handleError(w, "restore_listing", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"listing": listing})
}

func (h *ListingHandler) handleError(w http.ResponseWriter, operation string, err error) {
	var validation *catalog.ValidationError
	if errors.As(err, &validation) {
		writeValidationError(w, validation.Fields)
		return
	}

	switch {
	case errors.Is(err, catalog.ErrNotFound):
		writeError(w, http.StatusNotFound, "listing_not_found", "Listing tidak ditemukan")

	case errors.Is(err, catalog.ErrForbidden):
		writeError(w, http.StatusForbidden, "forbidden", "Anda bukan pemilik listing ini")

	case errors.Is(err, catalog.ErrNotVerified):
		writeError(w, http.StatusForbidden, "ktm_not_verified", "Akun harus terverifikasi KTM untuk mengelola listing")

	case errors.Is(err, catalog.ErrInvalidTransition):
		writeError(w, http.StatusConflict, "listing_invalid_transition", "Status listing tidak memungkinkan aksi ini")

	case errors.Is(err, catalog.ErrPhotoLimit):
		writeError(w, http.StatusConflict, "photo_limit", "Maksimal 3 foto per listing")

	case errors.Is(err, catalog.ErrListingInUse):
		writeError(w, http.StatusConflict, "listing_in_use", "Listing sudah dipakai transaksi dan tidak bisa dihapus")

	case errors.Is(err, catalog.ErrBlockOverlap):
		writeError(w, http.StatusConflict, "block_overlap", "Periode blokir bertabrakan dengan blokir atau rental lain")

	case errors.Is(err, catalog.ErrInvalidFile):
		writeValidationError(w, map[string]string{"file": "File foto wajib diisi"})

	case errors.Is(err, catalog.ErrUnsupportedType):
		writeValidationError(w, map[string]string{"file": "Format file harus JPEG atau PNG"})

	case errors.Is(err, catalog.ErrFileTooLarge):
		writeError(w, http.StatusRequestEntityTooLarge, "file_too_large", "Ukuran file melebihi 5 MB")

	case errors.Is(err, catalog.ErrInvalidReference):
		writeValidationError(w, map[string]string{
			"category_id":        "Kategori tidak ditemukan",
			"campus_location_id": "Lokasi tidak ditemukan",
		})

	case errors.Is(err, catalog.ErrInvalidStatus):
		writeValidationError(w, map[string]string{"status": "Status harus active atau taken_down"})

	default:
		slog.Error("kesalahan katalog", "operation", operation, "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "Terjadi kesalahan pada server")
	}
}
