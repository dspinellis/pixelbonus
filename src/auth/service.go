package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha512"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnverified         = errors.New("account not verified")
	ErrTokenInvalid       = errors.New("token invalid")
	ErrTokenExpired       = errors.New("token expired")
)

type Mailer interface {
	Send(context.Context, string, string, string) error
}

type Logger interface {
	Log(string, string, map[string]string)
}

type Service struct {
	DB      *sql.DB
	Mailer  Mailer
	Logger  Logger
	BaseURL string
	Now     func() time.Time
	Random  func([]byte) (int, error)
	Bcrypt  func([]byte, int) ([]byte, error)
	Compare func([]byte, []byte) error
}

type RegisterInput struct {
	FullName string
	Email    string
	Password string
}

type RegistrationResult struct {
	InstructorID int64
	Token        string
}

type ConfirmResult struct {
	AlreadyConfirmed bool
}

type InstructorIdentity struct {
	ID       int64
	FullName string
	Email    string
}

type PasswordResetResult struct {
	Token string
	Sent  bool
}

type ResetResult struct{}

func (s *Service) init() {
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Random == nil {
		s.Random = rand.Read
	}
	if s.Bcrypt == nil {
		s.Bcrypt = bcrypt.GenerateFromPassword
	}
	if s.Compare == nil {
		s.Compare = bcrypt.CompareHashAndPassword
	}
}

func (s *Service) RegisterInstructor(ctx context.Context, input RegisterInput) (RegistrationResult, error) {
	s.init()
	name := strings.TrimSpace(input.FullName)
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if len(name) == 0 || len(name) > 120 {
		return RegistrationResult{}, errors.New("invalid full name")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return RegistrationResult{}, errors.New("invalid email")
	}
	if len(input.Password) < 8 {
		return RegistrationResult{}, errors.New("invalid password")
	}
	hash, err := s.Bcrypt([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return RegistrationResult{}, err
	}
	token, err := nonce(s.Random, 24)
	if err != nil {
		return RegistrationResult{}, err
	}
	now := s.Now().UTC().Format(time.RFC3339)
	exp := s.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return RegistrationResult{}, err
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, "INSERT INTO Users(username,username_canonical,email,email_canonical,enabled,salt,password,locked,expired,roles,credentials_expired,name,grade_multiplier,max_grade,min_grade,locale) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)", email, email, email, email, false, "", string(hash), false, false, `a:0:{}`, false, name, 1, 10, 0, "en")
	if err != nil {
		return RegistrationResult{}, err
	}
	id, _ := res.LastInsertId()
	if _, err := tx.ExecContext(ctx, "INSERT INTO email_verification_tokens(token,instructor_id,expires_at,consumed_at,created_at) VALUES(?,?,?,?,?)", token, id, exp, nil, now); err != nil {
		return RegistrationResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return RegistrationResult{}, err
	}
	if s.Mailer != nil {
		body := fmt.Sprintf("%s/instructor/register/confirm?token=%s", strings.TrimRight(s.BaseURL, "/"), token)
		_ = s.Mailer.Send(ctx, email, "Verify your PixelBonus account", body)
	}
	if s.Logger != nil {
		s.Logger.Log("INFO", "register_instructor", map[string]string{"email": email})
	}
	return RegistrationResult{InstructorID: id, Token: token}, nil
}

func (s *Service) ConfirmInstructorEmail(ctx context.Context, token string) (ConfirmResult, error) {
	s.init()
	var instructorID int64
	var expiresAt string
	var consumedAt sql.NullString
	err := s.DB.QueryRowContext(ctx, "SELECT instructor_id, expires_at, consumed_at FROM email_verification_tokens WHERE token = ?", token).Scan(&instructorID, &expiresAt, &consumedAt)
	if err != nil {
		return ConfirmResult{}, ErrTokenInvalid
	}
	if consumedAt.Valid {
		return ConfirmResult{AlreadyConfirmed: true}, nil
	}
	exp, _ := time.Parse(time.RFC3339, expiresAt)
	if s.Now().UTC().After(exp) {
		return ConfirmResult{}, ErrTokenExpired
	}
	now := s.Now().UTC().Format(time.RFC3339)
	if _, err := s.DB.ExecContext(ctx, "UPDATE Users SET enabled = 1 WHERE id = ?", instructorID); err != nil {
		return ConfirmResult{}, err
	}
	if _, err := s.DB.ExecContext(ctx, "UPDATE email_verification_tokens SET consumed_at = ? WHERE token = ? AND consumed_at IS NULL", now, token); err != nil {
		return ConfirmResult{}, err
	}
	return ConfirmResult{}, nil
}

func (s *Service) Login(ctx context.Context, email string, password string) (InstructorIdentity, error) {
	s.init()
	email = strings.ToLower(strings.TrimSpace(email))
	var ident InstructorIdentity
	var hash, salt, surname string
	var enabled, locked, expired, credentialsExpired bool
	var expires, credentialsExpire sql.NullString
	err := s.DB.QueryRowContext(ctx, "SELECT id, COALESCE(name, username), COALESCE(surname, ''), email, password, salt, enabled, locked, expired, credentials_expired, expires_at, credentials_expire_at FROM Users WHERE email_canonical = ?", email).Scan(&ident.ID, &ident.FullName, &surname, &ident.Email, &hash, &salt, &enabled, &locked, &expired, &credentialsExpired, &expires, &credentialsExpire)
	if err != nil {
		return InstructorIdentity{}, ErrInvalidCredentials
	}
	if locked || expired || credentialsExpired || dateExpired(expires, s.Now()) || dateExpired(credentialsExpire, s.Now()) {
		return InstructorIdentity{}, ErrInvalidCredentials
	}
	if !enabled {
		return InstructorIdentity{}, ErrUnverified
	}
	if err := s.comparePassword(hash, salt, password); err != nil {
		return InstructorIdentity{}, ErrInvalidCredentials
	}
	ident.FullName = strings.TrimSpace(ident.FullName + " " + surname)
	return ident, nil
}

func (s *Service) CreatePasswordReset(ctx context.Context, email string) (PasswordResetResult, error) {
	s.init()
	email = strings.ToLower(strings.TrimSpace(email))
	var instructorID int64
	if err := s.DB.QueryRowContext(ctx, "SELECT id FROM Users WHERE email_canonical = ?", email).Scan(&instructorID); err != nil {
		if err == sql.ErrNoRows {
			return PasswordResetResult{Sent: false}, nil
		}
		return PasswordResetResult{}, err
	}
	token, err := nonce(s.Random, 24)
	if err != nil {
		return PasswordResetResult{}, err
	}
	now := s.Now().UTC().Format(time.RFC3339)
	exp := s.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	if _, err := s.DB.ExecContext(ctx, "INSERT INTO password_reset_tokens(token,instructor_id,expires_at,consumed_at,created_at) VALUES(?,?,?,?,?)", token, instructorID, exp, nil, now); err != nil {
		return PasswordResetResult{}, err
	}
	if s.Mailer != nil {
		body := fmt.Sprintf("%s/instructor/password/reset?token=%s", strings.TrimRight(s.BaseURL, "/"), token)
		_ = s.Mailer.Send(ctx, email, "Reset your PixelBonus password", body)
	}
	return PasswordResetResult{Token: token, Sent: true}, nil
}

func (s *Service) ResetPassword(ctx context.Context, token string, newPassword string) (ResetResult, error) {
	s.init()
	if len(newPassword) < 8 {
		return ResetResult{}, errors.New("invalid password")
	}
	var instructorID int64
	var expiresAt string
	var consumedAt sql.NullString
	if err := s.DB.QueryRowContext(ctx, "SELECT instructor_id, expires_at, consumed_at FROM password_reset_tokens WHERE token = ?", token).Scan(&instructorID, &expiresAt, &consumedAt); err != nil {
		return ResetResult{}, ErrTokenInvalid
	}
	if consumedAt.Valid {
		return ResetResult{}, ErrTokenInvalid
	}
	exp, _ := time.Parse(time.RFC3339, expiresAt)
	if s.Now().UTC().After(exp) {
		return ResetResult{}, ErrTokenExpired
	}
	hash, err := s.Bcrypt([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return ResetResult{}, err
	}
	now := s.Now().UTC().Format(time.RFC3339)
	if _, err := s.DB.ExecContext(ctx, "UPDATE Users SET password = ?, salt = '' WHERE id = ?", string(hash), instructorID); err != nil {
		return ResetResult{}, err
	}
	if _, err := s.DB.ExecContext(ctx, "UPDATE password_reset_tokens SET consumed_at = ? WHERE token = ?", now, token); err != nil {
		return ResetResult{}, err
	}
	return ResetResult{}, nil
}

func (s *Service) ChangePassword(ctx context.Context, instructorID int64, currentPassword string, newPassword string) error {
	s.init()
	if len(newPassword) < 8 {
		return errors.New("invalid password")
	}
	var hash, salt string
	if err := s.DB.QueryRowContext(ctx, "SELECT password, salt FROM Users WHERE id = ?", instructorID).Scan(&hash, &salt); err != nil {
		return ErrInvalidCredentials
	}
	if err := s.comparePassword(hash, salt, currentPassword); err != nil {
		return ErrInvalidCredentials
	}
	newHash, err := s.Bcrypt([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	_, err = s.DB.ExecContext(ctx, "UPDATE Users SET password = ?, salt = '' WHERE id = ?", string(newHash), instructorID)
	return err
}

func nonce(random func([]byte) (int, error), size int) (string, error) {
	buf := make([]byte, size)
	if _, err := random(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Symfony MessageDigestPasswordEncoder defaults: SHA-512, 5000 iterations.
// Other legacy encoder configurations require a password reset.
func (s *Service) comparePassword(hash, salt, password string) error {
	if strings.HasPrefix(hash, "$2") {
		return s.Compare([]byte(hash), []byte(password))
	}
	if len(password) > 4096 {
		return ErrInvalidCredentials
	}
	salted := password
	if salt != "" {
		salted += "{" + salt + "}"
	}
	digest := sha512.Sum512([]byte(salted))
	for i := 1; i < 5000; i++ {
		h := sha512.New()
		h.Write(digest[:])
		h.Write([]byte(salted))
		copy(digest[:], h.Sum(nil))
	}
	if subtle.ConstantTimeCompare([]byte(hash), []byte(base64.StdEncoding.EncodeToString(digest[:]))) == 1 || subtle.ConstantTimeCompare([]byte(hash), []byte(hex.EncodeToString(digest[:]))) == 1 {
		return nil
	}
	return ErrInvalidCredentials
}

func dateExpired(value sql.NullString, now time.Time) bool {
	if !value.Valid {
		return false
	}
	expiry, err := time.Parse("2006-01-02 15:04:05", value.String)
	return err != nil || !now.UTC().Before(expiry)
}
