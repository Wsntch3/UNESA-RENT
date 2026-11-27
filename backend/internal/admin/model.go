package admin

import "time"

type Report struct {
	ID             string     `json:"id"`
	ReporterID     string     `json:"reporter_id"`
	ReporterName   string     `json:"reporter_name"`
	RentalID       *string    `json:"rental_id,omitempty"`
	ListingID      *string    `json:"listing_id,omitempty"`
	Type           string     `json:"type"`
	Subject        string     `json:"subject"`
	Body           string     `json:"body"`
	Status         string     `json:"status"`
	ResolutionNote *string    `json:"resolution_note,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type ReportInput struct {
	Type      string `json:"type"`
	Subject   string `json:"subject"`
	Body      string `json:"body"`
	RentalID  string `json:"rental_id,omitempty"`
	ListingID string `json:"listing_id,omitempty"`
}

type ReportFilter struct {
	Status string `json:"status"`
	Type   string `json:"type"`
	Limit  int    `json:"limit"`
	Offset int    `json:"offset"`
}

type ReportPage struct {
	Items  []Report `json:"items"`
	Total  int      `json:"total"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}

type AdminUser struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	NIM         string     `json:"nim,omitempty"`
	IsVerified  bool       `json:"is_verified"`
	CreatedAt   time.Time  `json:"created_at"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

type UserFilter struct {
	Query    string `json:"q"`
	Role     string `json:"role"`
	Status   string `json:"status"`
	Verified string `json:"verified"`
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}

type UserPage struct {
	Items  []AdminUser `json:"items"`
	Total  int         `json:"total"`
	Limit  int         `json:"limit"`
	Offset int         `json:"offset"`
}

type AuditEntry struct {
	ID         string    `json:"id"`
	ActorID    string    `json:"actor_id"`
	ActorName  string    `json:"actor_name,omitempty"`
	Action     string    `json:"action"`
	TargetType string    `json:"target_type"`
	TargetID   string    `json:"target_id"`
	Reason     string    `json:"reason,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AuditFilter struct {
	ActorID string `json:"actor_id"`
	Action  string `json:"action"`
	From    string `json:"from"`
	To      string `json:"to"`
	Limit   int    `json:"limit"`
	Offset  int    `json:"offset"`
}

type AuditPage struct {
	Items  []AuditEntry `json:"items"`
	Total  int          `json:"total"`
	Limit  int          `json:"limit"`
	Offset int          `json:"offset"`
}

type Stats struct {
	TotalUsers      int            `json:"total_users"`
	VerifiedUsers   int            `json:"verified_users"`
	ActiveListings  int            `json:"active_listings"`
	OpenReports     int            `json:"open_reports"`
	TotalDocuments  int            `json:"total_documents"`
	StorageBytes    int64          `json:"storage_bytes"`
	RentalsByStatus map[string]int `json:"rentals_by_status"`
}
