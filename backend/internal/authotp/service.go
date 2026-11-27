package authotp

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"unesarent/backend/internal/email"
)

type Purpose string

const (
	PurposeRegister      Purpose = "register"
	PurposeResetPassword Purpose = "reset_password"
	PurposeAdminLogin    Purpose = "admin_login"
)

var (
	ErrTooManyRequests = errors.New("terlalu banyak permintaan kode")
	ErrInvalidCode     = errors.New("kode verifikasi salah")
	ErrExpired         = errors.New("kode verifikasi sudah kedaluwarsa")
	ErrInvalidTicket   = errors.New("token verifikasi tidak valid")
)

const (
	emailWindow = 10 * time.Minute
	ipWindow    = time.Hour
	emailLimit  = 3
	ipLimit     = 10
)

type Service struct {
	store       Store
	mail        email.Sender
	pepper      []byte
	ttl         time.Duration
	maxAttempts int
	now         func() time.Time
}

func NewService(store Store, mail email.Sender, pepper []byte, ttl time.Duration, maxAttempts int) *Service {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	if maxAttempts < 1 || maxAttempts > 10 {
		maxAttempts = 5
	}
	return &Service{
		store:       store,
		mail:        mail,
		pepper:      pepper,
		ttl:         ttl,
		maxAttempts: maxAttempts,
		now:         func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Request(ctx context.Context, addr string, purpose Purpose, ip string) error {
	addr = normalizeEmail(addr)
	if !validPurpose(purpose) {
		return ErrInvalidCode
	}
	if err := s.checkRate(ctx, addr, purpose, ip); err != nil {
		return err
	}

	code, err := randomCode()
	if err != nil {
		return err
	}
	rec := NewRecord{
		Email:       addr,
		Purpose:     purpose,
		CodeHash:    s.hashCode(code),
		MaxAttempts: s.maxAttempts,
		ExpiresAt:   s.now().Add(s.ttl),
		RequestIP:   ip,
	}
	if _, err := s.store.Insert(ctx, rec); err != nil {
		return err
	}
	return s.mail.Send(ctx, addr, email.OTPSubject(), email.OTPBody(code, int(s.ttl.Minutes())))
}

func (s *Service) Verify(ctx context.Context, addr string, purpose Purpose, code string) (string, error) {
	addr = normalizeEmail(addr)
	code = strings.TrimSpace(code)
	if code == "" {
		return "", ErrInvalidCode
	}

	rec, err := s.store.LatestActive(ctx, addr, purpose)
	if err != nil {
		if errors.Is(err, ErrNoActive) {
			return "", ErrExpired
		}
		return "", err
	}

	if !hmac.Equal([]byte(rec.CodeHash), []byte(s.hashCode(code))) {
		attempts, incErr := s.store.IncrementAttempts(ctx, rec.ID)
		if incErr != nil {
			return "", incErr
		}
		if attempts >= rec.MaxAttempts {
			_ = s.store.Consume(ctx, rec.ID)
		}
		return "", ErrInvalidCode
	}

	ticket, err := randomToken()
	if err != nil {
		return "", err
	}
	if err := s.store.MarkVerified(ctx, rec.ID, hashToken(ticket)); err != nil {
		return "", err
	}
	return ticket, nil
}

func (s *Service) ConsumeTicket(ctx context.Context, addr string, purpose Purpose, ticket string) error {
	ticket = strings.TrimSpace(ticket)
	if ticket == "" {
		return ErrInvalidTicket
	}
	ok, err := s.store.ConsumeByChallenge(ctx, normalizeEmail(addr), purpose, hashToken(ticket))
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidTicket
	}
	return nil
}

func (s *Service) CreateChallenge(ctx context.Context, addr, userID, ip string) (string, error) {
	addr = normalizeEmail(addr)
	if err := s.checkRate(ctx, addr, PurposeAdminLogin, ip); err != nil {
		return "", err
	}

	code, err := randomCode()
	if err != nil {
		return "", err
	}
	challenge, err := randomToken()
	if err != nil {
		return "", err
	}
	challengeHash := hashToken(challenge)

	rec := NewRecord{
		Email:         addr,
		UserID:        &userID,
		Purpose:       PurposeAdminLogin,
		CodeHash:      s.hashCode(code),
		ChallengeHash: &challengeHash,
		MaxAttempts:   s.maxAttempts,
		ExpiresAt:     s.now().Add(s.ttl),
		RequestIP:     ip,
	}
	if _, err := s.store.Insert(ctx, rec); err != nil {
		return "", err
	}
	if err := s.mail.Send(ctx, addr, email.OTPSubject(), email.OTPBody(code, int(s.ttl.Minutes()))); err != nil {
		return "", err
	}
	return challenge, nil
}

func (s *Service) VerifyChallenge(ctx context.Context, challenge, code string) (string, error) {
	rec, err := s.store.FindByChallenge(ctx, hashToken(strings.TrimSpace(challenge)))
	if err != nil {
		if errors.Is(err, ErrNoActive) {
			return "", ErrExpired
		}
		return "", err
	}
	if !hmac.Equal([]byte(rec.CodeHash), []byte(s.hashCode(strings.TrimSpace(code)))) {
		attempts, incErr := s.store.IncrementAttempts(ctx, rec.ID)
		if incErr != nil {
			return "", incErr
		}
		if attempts >= rec.MaxAttempts {
			_ = s.store.Consume(ctx, rec.ID)
		}
		return "", ErrInvalidCode
	}
	if err := s.store.Consume(ctx, rec.ID); err != nil {
		return "", err
	}
	return rec.Email, nil
}

func (s *Service) checkRate(ctx context.Context, addr string, purpose Purpose, ip string) error {
	n, err := s.store.CountRecent(ctx, addr, purpose, emailWindow)
	if err != nil {
		return err
	}
	if n >= emailLimit {
		return ErrTooManyRequests
	}
	n, err = s.store.CountRecentIP(ctx, ip, ipWindow)
	if err != nil {
		return err
	}
	if n >= ipLimit {
		return ErrTooManyRequests
	}
	return nil
}

func (s *Service) hashCode(code string) string {
	mac := hmac.New(sha256.New, s.pepper)
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func newID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func normalizeEmail(addr string) string {
	return strings.ToLower(strings.TrimSpace(addr))
}

func validPurpose(p Purpose) bool {
	switch p {
	case PurposeRegister, PurposeResetPassword, PurposeAdminLogin:
		return true
	}
	return false
}
