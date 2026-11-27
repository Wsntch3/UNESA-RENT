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
	"unesarent/backend/internal/authotp"
)

const (
	minPasswordLength = 8
	maxPasswordLength = 72
	maxNameLength     = 80
	maxEmailLength    = 254
	nimLength         = 11
)

var nimPattern = regexp.MustCompile(fmt.Sprintf(`^[0-9]{%d}$`, nimLength))

var (
	ErrInvalidCredentials = errors.New("email atau password salah")
	ErrIdentityVerified   = errors.New("identitas sudah terverifikasi")
)

type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return "data tidak valid"
}

type AuthResult struct {
	User        PublicUser
	Token       string
	ExpiresAt   time.Time
	OTPRequired bool
	Challenge   string
}

type Service struct {
	store         Store
	tokens        *auth.TokenIssuer
	otp           *authotp.Service
	allowedDomain []string
	adminLoginOTP bool
}

func NewService(store Store, tokens *auth.TokenIssuer, otp *authotp.Service, allowedDomains []string, adminLoginOTP bool) *Service {
	return &Service{
		store:         store,
		tokens:        tokens,
		otp:           otp,
		allowedDomain: allowedDomains,
		adminLoginOTP: adminLoginOTP,
	}
}

type RegisterInput struct {
	Name              string `json:"name"`
	Email             string `json:"email"`
	Password          string `json:"password"`
	NIM               string `json:"nim"`
	VerificationToken string `json:"verification_token"`
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
	if !s.isAllowedStudentEmail(in.Email) {
		return AuthResult{}, &ValidationError{Fields: map[string]string{
			"email": "Gunakan email resmi mahasiswa @mhs.unesa.ac.id",
		}}
	}

	if err := s.otp.ConsumeTicket(ctx, in.Email, authotp.PurposeRegister, in.VerificationToken); err != nil {
		return AuthResult{}, &ValidationError{Fields: map[string]string{
			"verification_token": "Kode verifikasi tidak valid atau sudah kedaluwarsa. Minta kode baru.",
		}}
	}

	status, err := s.store.FindNIMStatus(ctx, in.NIM)
	if err != nil {
		return AuthResult{}, fmt.Errorf("memeriksa NIM: %w", err)
	}
	if status.Exists && status.Verified {
		return AuthResult{}, ErrIdentityVerified
	}
	if status.Exists {
		return AuthResult{}, ErrNIMTaken
	}

	if _, err := s.store.FindByEmail(ctx, in.Email); err == nil {
		return AuthResult{}, ErrEmailTaken
	} else if !errors.Is(err, ErrNotFound) {
		return AuthResult{}, fmt.Errorf("memeriksa email: %w", err)
	}

	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC()
	newUser := User{
		Name:            in.Name,
		Email:           in.Email,
		PasswordHash:    hash,
		Role:            RoleMahasiswa,
		NIM:             in.NIM,
		Status:          StatusActive,
		EmailVerifiedAt: &now,
	}

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

	if found.Status != StatusActive {
		return AuthResult{}, ErrAccountSuspended
	}

	if found.Role == RoleAdminDev && s.adminLoginOTP {
		challenge, err := s.otp.CreateChallenge(ctx, found.Email, found.ID, "")
		if err != nil {
			return AuthResult{}, err
		}
		return AuthResult{
			OTPRequired: true,
			Challenge:   challenge,
			ExpiresAt:   time.Now().UTC().Add(15 * time.Minute),
		}, nil
	}

	if err := s.store.MarkLogin(ctx, found.ID); err != nil {
		return AuthResult{}, fmt.Errorf("mencatat login: %w", err)
	}
	return s.buildResult(found)
}

func (s *Service) LoginOTP(ctx context.Context, challenge, code string) (AuthResult, error) {
	email, err := s.otp.VerifyChallenge(ctx, challenge, code)
	if err != nil {
		return AuthResult{}, err
	}
	found, err := s.store.FindByEmail(ctx, email)
	if err != nil {
		return AuthResult{}, err
	}
	if found.Role != RoleAdminDev || found.Status != StatusActive {
		return AuthResult{}, ErrAccountSuspended
	}
	if err := s.store.MarkLogin(ctx, found.ID); err != nil {
		return AuthResult{}, fmt.Errorf("mencatat login: %w", err)
	}
	return s.buildResult(found)
}

func (s *Service) RequestOTP(ctx context.Context, email string, purpose authotp.Purpose, ip string) error {
	email = normalizeEmail(email)

	if purpose == authotp.PurposeRegister && !s.isAllowedStudentEmail(email) {
		return nil
	}
	return s.otp.Request(ctx, email, purpose, ip)
}

func (s *Service) VerifyOTP(ctx context.Context, email string, purpose authotp.Purpose, code string) (string, error) {
	return s.otp.Verify(ctx, email, purpose, code)
}

func (s *Service) ForgotPassword(ctx context.Context, email, ip string) error {
	email = normalizeEmail(email)
	if _, err := s.store.FindByEmail(ctx, email); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		return err
	}
	return s.otp.Request(ctx, email, authotp.PurposeResetPassword, ip)
}

func (s *Service) ResetPassword(ctx context.Context, email, ticket, newPassword string) error {
	email = normalizeEmail(email)
	if fields := validatePassword(newPassword); len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	if err := s.otp.ConsumeTicket(ctx, email, authotp.PurposeResetPassword, ticket); err != nil {
		return &ValidationError{Fields: map[string]string{
			"ticket": "Kode verifikasi tidak valid atau sudah kedaluwarsa. Minta kode baru.",
		}}
	}
	found, err := s.store.FindByEmail(ctx, email)
	if err != nil {
		return ErrNotFound
	}
	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.store.UpdatePassword(ctx, found.ID, hash)
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

func (s *Service) isAllowedStudentEmail(email string) bool {
	email = normalizeEmail(email)
	at := strings.LastIndex(email, "@")
	if at < 0 || at == len(email)-1 {
		return false
	}
	domain := email[at+1:]
	for _, allowed := range s.allowedDomain {
		allowed = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(allowed)), "@")
		if allowed != "" && domain == allowed {
			return true
		}
	}
	return false
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

	for key, message := range validatePassword(in.Password) {
		fields[key] = message
	}

	switch {
	case in.NIM == "":
		fields["nim"] = "NIM wajib diisi"
	case !nimPattern.MatchString(in.NIM):
		fields["nim"] = fmt.Sprintf("NIM harus berupa %d digit angka", nimLength)
	}
	if strings.TrimSpace(in.VerificationToken) == "" {
		fields["verification_token"] = "Verifikasi email terlebih dahulu"
	}

	return fields
}

func validatePassword(password string) map[string]string {
	fields := make(map[string]string)
	pwLen := len(password)
	switch {
	case pwLen == 0:
		fields["password"] = "Password wajib diisi"
	case pwLen < minPasswordLength:
		fields["password"] = fmt.Sprintf("Password minimal %d karakter", minPasswordLength)
	case pwLen > maxPasswordLength:
		fields["password"] = fmt.Sprintf("Password maksimal %d byte", maxPasswordLength)
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
