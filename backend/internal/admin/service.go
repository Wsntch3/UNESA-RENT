package admin

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrForbidden         = errors.New("akses ditolak")
	ErrInvalidActor      = errors.New("admin tidak ditemukan")
	ErrCannotSuspendSelf = errors.New("tidak bisa menangguhkan akun sendiri")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "validasi gagal" }

type Service struct {
	store *PostgresStore
}

func NewService(store *PostgresStore) *Service {
	return &Service{store: store}
}

func clamp(value, fallback, max int) int {
	if value <= 0 {
		return fallback
	}
	if value > max {
		return max
	}
	return value
}

func reasonFields(raw string) map[string]string {
	n := utf8.RuneCountInString(strings.TrimSpace(raw))
	if n < 10 || n > 1000 {
		return map[string]string{"reason": "Alasan harus 10-1000 karakter"}
	}
	return nil
}

func (s *Service) CreateReport(ctx context.Context, reporterID string, in ReportInput) (Report, error) {
	fields := map[string]string{}
	in.Type = strings.ToLower(strings.TrimSpace(in.Type))
	switch in.Type {
	case "technical", "safety", "fraud", "harassment", "other":
	default:
		fields["type"] = "Jenis laporan tidak valid"
	}
	in.Subject = strings.TrimSpace(in.Subject)
	if n := utf8.RuneCountInString(in.Subject); n < 3 || n > 150 {
		fields["subject"] = "Judul harus 3-150 karakter"
	}
	in.Body = strings.TrimSpace(in.Body)
	if n := utf8.RuneCountInString(in.Body); n < 10 || n > 5000 {
		fields["body"] = "Isi laporan harus 10-5000 karakter"
	}
	if len(fields) > 0 {
		return Report{}, &ValidationError{Fields: fields}
	}
	return s.store.CreateReport(ctx, reporterID, in)
}

func (s *Service) MyReports(ctx context.Context, userID string, limit, offset int) (ReportPage, error) {
	items, total, err := s.store.Reports(ctx, ReportFilter{Limit: clamp(limit, 20, 60), Offset: offset})
	if err != nil {
		return ReportPage{}, err
	}
	filtered := make([]Report, 0)
	for _, item := range items {
		if item.ReporterID == userID {
			filtered = append(filtered, item)
		}
	}
	// Hitung ulang total milik pengguna agar pagination tetap jujur.
	all, _, err := s.store.Reports(ctx, ReportFilter{Limit: 10000})
	if err != nil {
		return ReportPage{}, err
	}
	total = 0
	for _, item := range all {
		if item.ReporterID == userID {
			total++
		}
	}
	return ReportPage{Items: filtered, Total: total, Limit: clamp(limit, 20, 60), Offset: offset}, nil
}

func (s *Service) Report(ctx context.Context, id string) (Report, error) {
	return s.store.ReportByID(ctx, id)
}

func (s *Service) Reports(ctx context.Context, f ReportFilter) (ReportPage, error) {
	f.Status = strings.ToLower(strings.TrimSpace(f.Status))
	f.Type = strings.ToLower(strings.TrimSpace(f.Type))
	f.Limit = clamp(f.Limit, 20, 60)
	items, total, err := s.store.Reports(ctx, f)
	if err != nil {
		return ReportPage{}, err
	}
	return ReportPage{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

func (s *Service) ResolveReport(ctx context.Context, adminID, id, action, note string, ip, userAgent string) (Report, error) {
	if fields := reasonFields(note); fields != nil {
		return Report{}, &ValidationError{Fields: fields}
	}
	status := "resolved"
	if action == "reject" {
		status = "rejected"
	} else if action != "resolve" {
		return Report{}, ErrForbidden
	}
	report, err := s.store.ResolveReport(ctx, id, adminID, status, strings.TrimSpace(note))
	if err != nil {
		return Report{}, err
	}
	actionName := "resolve_report"
	title := "Laporan diselesaikan"
	if status == "rejected" {
		actionName = "reject_report"
		title = "Laporan ditolak"
	}
	_ = s.store.LogAction(ctx, adminID, actionName, "report", id, strings.TrimSpace(note), map[string]any{"ip": ip, "user_agent": userAgent})
	_ = s.store.Notify(ctx, report.ReporterID, title,
		"Laporan Anda \""+report.Subject+"\" telah "+map[string]string{"resolved": "diselesaikan", "rejected": "ditolak"}[status]+".",
		map[string]any{"report_id": id, "status": status})
	return report, nil
}

func (s *Service) Users(ctx context.Context, f UserFilter) (UserPage, error) {
	f.Limit = clamp(f.Limit, 20, 60)
	items, total, err := s.store.AdminUsers(ctx, f)
	if err != nil {
		return UserPage{}, err
	}
	return UserPage{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

func (s *Service) User(ctx context.Context, id string) (AdminUser, error) {
	return s.store.AdminUserByID(ctx, id)
}

func (s *Service) ExportUsers(ctx context.Context, f UserFilter) ([]AdminUser, error) {
	return s.store.ExportUsers(ctx, f)
}

func (s *Service) Suspend(ctx context.Context, adminID, id, reason, ip, userAgent string) error {
	if id == adminID {
		return ErrCannotSuspendSelf
	}
	if fields := reasonFields(reason); fields != nil {
		return &ValidationError{Fields: fields}
	}
	if err := s.store.SetUserStatus(ctx, id, "suspended"); err != nil {
		return err
	}
	return s.store.LogAction(ctx, adminID, "suspend_user", "user", id, strings.TrimSpace(reason), map[string]any{"ip": ip, "user_agent": userAgent})
}

func (s *Service) Reactivate(ctx context.Context, adminID, id, reason, ip, userAgent string) error {
	if fields := reasonFields(reason); fields != nil {
		return &ValidationError{Fields: fields}
	}
	if err := s.store.SetUserStatus(ctx, id, "active"); err != nil {
		return err
	}
	return s.store.LogAction(ctx, adminID, "reactivate_user", "user", id, strings.TrimSpace(reason), map[string]any{"ip": ip, "user_agent": userAgent})
}

func (s *Service) DeleteUser(ctx context.Context, adminID, id, reason, ip, userAgent string) error {
	if fields := reasonFields(reason); fields != nil {
		return &ValidationError{Fields: fields}
	}
	if err := s.store.DeleteUnverifiedUser(ctx, id); err != nil {
		return err
	}
	return s.store.LogAction(ctx, adminID, "delete_unverified_user", "user", id, strings.TrimSpace(reason), map[string]any{"ip": ip, "user_agent": userAgent})
}

func (s *Service) Audit(ctx context.Context, f AuditFilter) (AuditPage, error) {
	f.Limit = clamp(f.Limit, 20, 60)
	items, total, err := s.store.Audit(ctx, f)
	if err != nil {
		return AuditPage{}, err
	}
	return AuditPage{Items: items, Total: total, Limit: f.Limit, Offset: f.Offset}, nil
}

func (s *Service) Stats(ctx context.Context) (Stats, error) {
	return s.store.Stats(ctx)
}

func (s *Service) LogExport(ctx context.Context, adminID, targetType, reason, ip, userAgent string) {
	_ = s.store.LogAction(ctx, adminID, "export_data", targetType, adminID, reason,
		map[string]any{"ip": ip, "user_agent": userAgent})
}
