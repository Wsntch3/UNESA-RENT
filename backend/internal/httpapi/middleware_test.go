package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"unesarent/backend/internal/auth"
	"unesarent/backend/internal/user"
)

type fakeAuthStates map[string]user.AuthState

func (f fakeAuthStates) FindAuthState(_ context.Context, id string) (user.AuthState, error) {
	state, ok := f[id]
	if !ok {
		return user.AuthState{}, user.ErrNotFound
	}
	return state, nil
}

func TestRequireRole(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	tokens := auth.NewTokenIssuer(secret, time.Hour)

	issue := func(subject, role string, tokenVersion int) string {
		t.Helper()
		token, _, err := tokens.Issue(subject, subject+"@example.com", "User", role, tokenVersion)
		if err != nil {
			t.Fatalf("Issue() error = %v", err)
		}
		return token
	}

	states := fakeAuthStates{
		"admin-1":   {Role: user.RoleAdminDev, Status: user.StatusActive, TokenVersion: 0},
		"student-1": {Role: user.RoleMahasiswa, Status: user.StatusActive, TokenVersion: 0},
		"suspended": {Role: user.RoleAdminDev, Status: user.StatusSuspended, TokenVersion: 0},
		"stale":     {Role: user.RoleAdminDev, Status: user.StatusActive, TokenVersion: 1},
	}

	protected := RequireAuth(tokens, states)(RequireRole(string(user.RoleAdminDev))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	tests := []struct {
		name       string
		token      string
		wantStatus int
	}{
		{name: "tanpa token", token: "", wantStatus: http.StatusUnauthorized},
		{name: "admin aktif", token: issue("admin-1", string(user.RoleAdminDev), 0), wantStatus: http.StatusOK},
		{name: "role dari database bukan token", token: issue("admin-1", string(user.RoleMahasiswa), 0), wantStatus: http.StatusOK},
		{name: "mahasiswa", token: issue("student-1", string(user.RoleMahasiswa), 0), wantStatus: http.StatusForbidden},
		{name: "admin ditangguhkan", token: issue("suspended", string(user.RoleAdminDev), 0), wantStatus: http.StatusUnauthorized},
		{name: "token version lama", token: issue("stale", string(user.RoleAdminDev), 0), wantStatus: http.StatusUnauthorized},
		{name: "user tidak ada", token: issue("ghost", string(user.RoleAdminDev), 0), wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/admin/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", "Bearer "+tt.token)
			}

			rec := httptest.NewRecorder()
			protected.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
