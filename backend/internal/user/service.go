package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"unesarent/backend/internal/auth"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72
	maxNameLength     = 80
	maxEmailLength    = 254
	nimLength         = 11 // AA FF PPP J UUU
)

var nimPattern = regexp.MustCompile(fmt.Sprintf(`^[0-9]{%d}$`, nimLength))

var ErrInvalidCredentials = errors.New("email atau password salah")

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "data tidak valid"
}

type AuthResult struct {
	User      PublicUser
	Token     string
	ExpiresAt time.Time
}

type Service struct {
	store  Store
	tokens *auth.TokenIssuer
}

func NewService(store Store, tokens *auth.TokenIssuer) *Service {
	return &Service{store: store, tokens: tokens}
}

type RegisterInput struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	NIM      string `json:"nim"`
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Service) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = normalizeEmail(in.Email)
	in.NIM = strings.TrimSpace(in.NIM)

	if fields := validateRegister(in); len(fields) > 0 {
		return AuthResult{}, &ValidationError{Fields: fields}
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	newUser := User{
		Name:         in.Name,
		Email:        in.Email,
		PasswordHash: hash,
		Role:         RoleMahasiswa,
		NIM:          in.NIM,
		Status:       StatusActive,
	}

	// Store yang mengisi ID dan kolom turunan database (mis. study_program_code, token_version).
	created, err := s.store.Create(ctx, newUser)
	if err != nil {
		return AuthResult{}, err
	}

	return s.buildResult(created)
}

var dummyHash = func() string {
	h, _ := auth.HashPassword("dummy-password-untuk-timing")
	return h
}()

func (s *Service) Login(ctx context.Context, in LoginInput) (AuthResult, error) {
	email := normalizeEmail(in.Email)

	if email == "" || in.Password == "" {
		fields := make(map[string]string)
		if email == "" {
			fields["email"] = "Email wajib diisi"
		}
		if in.Password == "" {
			fields["password"] = "Password wajib diisi"
		}
		return AuthResult{}, &ValidationError{Fields: fields}
	}

	found, err := s.store.FindByEmail(ctx, email)
	if errors.Is(err, ErrNotFound) {
		auth.CheckPassword(dummyHash, in.Password)
		return AuthResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, fmt.Errorf("mencari user: %w", err)
	}

	if !auth.CheckPassword(found.PasswordHash, in.Password) {
		return AuthResult{}, ErrInvalidCredentials
	}

	// Status dicek setelah password benar agar status akun tidak bocor
	// ke orang yang tidak tahu passwordnya.
	if found.Status != StatusActive {
		return AuthResult{}, ErrAccountSuspended
	}

	return s.buildResult(found)
}

func (s *Service) Me(ctx context.Context, userID string) (PublicUser, error) {
	u, err := s.store.FindByID(ctx, userID)
	if err != nil {
		return PublicUser{}, err
	}
	return u.Public(), nil
}

func (s *Service) buildResult(u User) (AuthResult, error) {
	token, expiresAt, err := s.tokens.Issue(u.ID, u.Email, u.Name, string(u.Role), u.TokenVersion)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{
		User:      u.Public(),
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func validateRegister(in RegisterInput) map[string]string {
	fields := make(map[string]string)

	nameLen := utf8.RuneCountInString(in.Name)
	switch {
	case nameLen == 0:
		fields["name"] = "Nama wajib diisi"
	case nameLen < 2:
		fields["name"] = "Nama minimal 2 karakter"
	case nameLen > maxNameLength:
		fields["name"] = fmt.Sprintf("Nama maksimal %d karakter", maxNameLength)
	}

	switch {
	case in.Email == "":
		fields["email"] = "Email wajib diisi"
	case len(in.Email) > maxEmailLength:
		fields["email"] = "Email terlalu panjang"
	case !isValidEmail(in.Email):
		fields["email"] = "Format email tidak valid"
	}

	pwLen := len(in.Password)
	switch {
	case pwLen == 0:
		fields["password"] = "Password wajib diisi"
	case pwLen < minPasswordLength:
		fields["password"] = fmt.Sprintf("Password minimal %d karakter", minPasswordLength)
	case pwLen > maxPasswordLength:
		fields["password"] = fmt.Sprintf("Password maksimal %d byte", maxPasswordLength)
	}

	switch {
	case in.NIM == "":
		fields["nim"] = "NIM wajib diisi"
	case !nimPattern.MatchString(in.NIM):
		fields["nim"] = fmt.Sprintf("NIM harus berupa %d digit angka", nimLength)
	}

	return fields
}

func isValidEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	if err != nil {
		return false
	}
	return strings.EqualFold(addr.Address, email) && strings.Contains(email, "@")
}