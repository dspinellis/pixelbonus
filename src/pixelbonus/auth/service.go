package auth

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pixelbonus/src/pixelbonus/email"
)

var (
	ErrInvalidInput   = errors.New("invalid input")
	ErrDuplicateEmail = errors.New("duplicate email")
	ErrInvalidLogin   = errors.New("invalid login")
	ErrUnverified     = errors.New("account is not verified")
	ErrTokenInvalid   = errors.New("token is invalid")
	ErrNotFound       = errors.New("not found")
)

type Service struct {
	db      *sql.DB
	sender  email.Sender
	baseURL string
	clock   func() time.Time
	random  io.Reader
}

type Instructor struct {
	ID         int64
	FullName   string
	Email      string
	VerifiedAt sql.NullTime
	CreatedAt  time.Time
}

type Session struct {
	ID           string
	InstructorID int64
	CSRFToken    string
	CreatedAt    time.Time
}

type RegisterInstructorInput struct {
	FullName string
	Email    string
	Password string
}

func NewService(db *sql.DB, sender email.Sender, baseURL string, clock func() time.Time, random io.Reader) *Service {
	return &Service{db: db, sender: sender, baseURL: strings.TrimRight(baseURL, "/"), clock: clock, random: random}
}

func (s *Service) RegisterInstructor(ctx context.Context, input RegisterInstructorInput) (Instructor, error) {
	fullName := strings.TrimSpace(input.FullName)
	emailValue := strings.TrimSpace(strings.ToLower(input.Email))
	if fullName == "" || !validEmail(emailValue) || len(input.Password) < 8 {
		return Instructor{}, ErrInvalidInput
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return Instructor{}, err
	}
	now := s.clock().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Instructor{}, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO instructors (full_name, email, password_hash, verified_at, created_at)
		VALUES (?, ?, ?, NULL, ?)
	`, fullName, emailValue, string(hash), now)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return Instructor{}, ErrDuplicateEmail
		}
		return Instructor{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Instructor{}, err
	}
	nonce, err := randomHex(s.random, 16)
	if err != nil {
		return Instructor{}, err
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO verification_tokens (nonce, instructor_id, expires_at, created_at, consumed_at)
		VALUES (?, ?, ?, ?, NULL)
	`, nonce, id, now.Add(24*time.Hour), now)
	if err != nil {
		return Instructor{}, err
	}
	if err := tx.Commit(); err != nil {
		return Instructor{}, err
	}

	link := s.baseURL + "/instructor/verify?token=" + nonce
	body := "Verify your PixelBonus account:\n\n" + link + "\n"
	if err := s.sender.Send(emailValue, "Verify your PixelBonus account", body); err != nil {
		return Instructor{}, err
	}
	return Instructor{ID: id, FullName: fullName, Email: emailValue, CreatedAt: now}, nil
}

func (s *Service) VerifyInstructor(ctx context.Context, nonce string) error {
	now := s.clock().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var instructorID int64
	var expires time.Time
	var consumed sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT instructor_id, expires_at, consumed_at
		FROM verification_tokens
		WHERE nonce = ?
	`, nonce).Scan(&instructorID, &expires, &consumed)
	if err == sql.ErrNoRows {
		return ErrTokenInvalid
	}
	if err != nil {
		return err
	}
	if consumed.Valid || now.After(expires) {
		return ErrTokenInvalid
	}
	if _, err := tx.ExecContext(ctx, `UPDATE instructors SET verified_at = ? WHERE id = ?`, now, instructorID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE verification_tokens SET consumed_at = ? WHERE nonce = ?`, now, nonce); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) Login(ctx context.Context, emailValue string, password string) (Session, error) {
	var inst Instructor
	var passwordHash string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, full_name, email, verified_at, created_at, password_hash
		FROM instructors
		WHERE email = ?
	`, strings.ToLower(strings.TrimSpace(emailValue))).Scan(&inst.ID, &inst.FullName, &inst.Email, &inst.VerifiedAt, &inst.CreatedAt, &passwordHash)
	if err == sql.ErrNoRows {
		return Session{}, ErrInvalidLogin
	}
	if err != nil {
		return Session{}, err
	}
	if !inst.VerifiedAt.Valid {
		return Session{}, ErrUnverified
	}
	if bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password)) != nil {
		return Session{}, ErrInvalidLogin
	}
	sessionID, err := randomHex(s.random, 32)
	if err != nil {
		return Session{}, err
	}
	csrf, err := randomHex(s.random, 32)
	if err != nil {
		return Session{}, err
	}
	session := Session{ID: sessionID, InstructorID: inst.ID, CSRFToken: csrf, CreatedAt: s.clock().UTC()}
	_, err = s.db.ExecContext(ctx, `
		INSERT INTO sessions (id, instructor_id, csrf_token, created_at)
		VALUES (?, ?, ?, ?)
	`, session.ID, session.InstructorID, session.CSRFToken, session.CreatedAt)
	if err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

func (s *Service) RequestPasswordReset(ctx context.Context, emailValue string) error {
	emailValue = strings.ToLower(strings.TrimSpace(emailValue))
	var instructorID int64
	var verified sql.NullTime
	err := s.db.QueryRowContext(ctx, `SELECT id, verified_at FROM instructors WHERE email = ?`, emailValue).Scan(&instructorID, &verified)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	if !verified.Valid {
		return nil
	}
	now := s.clock().UTC()
	nonce, err := randomHex(s.random, 16)
	if err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO password_reset_tokens (nonce, instructor_id, expires_at, created_at, consumed_at)
		VALUES (?, ?, ?, ?, NULL)
	`, nonce, instructorID, now.Add(time.Hour), now); err != nil {
		return err
	}
	link := s.baseURL + "/instructor/password/reset?token=" + nonce
	return s.sender.Send(emailValue, "Reset your PixelBonus password", "Reset your PixelBonus password:\n\n"+link+"\n")
}

func (s *Service) ResetPassword(ctx context.Context, nonce string, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrInvalidInput
	}
	now := s.clock().UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var instructorID int64
	var expires time.Time
	var consumed sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT instructor_id, expires_at, consumed_at
		FROM password_reset_tokens
		WHERE nonce = ?
	`, nonce).Scan(&instructorID, &expires, &consumed)
	if err == sql.ErrNoRows {
		return ErrTokenInvalid
	}
	if err != nil {
		return err
	}
	if consumed.Valid || now.After(expires) {
		return ErrTokenInvalid
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE instructors SET password_hash = ? WHERE id = ?`, string(hash), instructorID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE password_reset_tokens SET consumed_at = ? WHERE nonce = ?`, now, nonce); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) ChangePassword(ctx context.Context, instructorID int64, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrInvalidInput
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `UPDATE instructors SET password_hash = ? WHERE id = ?`, string(hash), instructorID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) InstructorBySession(ctx context.Context, sessionID string) (Instructor, Session, error) {
	var inst Instructor
	var sess Session
	err := s.db.QueryRowContext(ctx, `
		SELECT i.id, i.full_name, i.email, i.verified_at, i.created_at, s.id, s.instructor_id, s.csrf_token, s.created_at
		FROM sessions s
		JOIN instructors i ON i.id = s.instructor_id
		WHERE s.id = ?
	`, sessionID).Scan(&inst.ID, &inst.FullName, &inst.Email, &inst.VerifiedAt, &inst.CreatedAt, &sess.ID, &sess.InstructorID, &sess.CSRFToken, &sess.CreatedAt)
	if err == sql.ErrNoRows {
		return Instructor{}, Session{}, ErrNotFound
	}
	if err != nil {
		return Instructor{}, Session{}, err
	}
	return inst, sess, nil
}

func validEmail(value string) bool {
	if value == "" {
		return false
	}
	_, err := mail.ParseAddress(value)
	return err == nil
}

func randomHex(r io.Reader, size int) (string, error) {
	buf := make([]byte, size)
	if _, err := io.ReadFull(r, buf); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
