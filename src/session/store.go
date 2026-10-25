package session

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"
)

var ErrInvalidSession = errors.New("invalid session")

type SessionData struct {
	ID           string
	InstructorID int64
}

type Store struct {
	DB     *sql.DB
	Now    func() time.Time
	Random func([]byte) (int, error)
}

func (s *Store) init() {
	if s.Now == nil {
		s.Now = time.Now
	}
	if s.Random == nil {
		s.Random = rand.Read
	}
}

func (s *Store) Create(ctx context.Context, instructorID int64, now time.Time) (SessionData, error) {
	s.init()
	buf := make([]byte, 32)
	if _, err := s.Random(buf); err != nil {
		return SessionData{}, err
	}
	id := base64.RawURLEncoding.EncodeToString(buf)
	issued := now.UTC()
	if _, err := s.DB.ExecContext(ctx, "INSERT INTO sessions(id,instructor_id,issued_at,last_seen_at,expires_at,absolute_expires_at) VALUES(?,?,?,?,?,?)",
		id, instructorID, issued.Format(time.RFC3339), issued.Format(time.RFC3339), issued.Add(12*time.Hour).Format(time.RFC3339), issued.Add(7*24*time.Hour).Format(time.RFC3339)); err != nil {
		return SessionData{}, err
	}
	return SessionData{ID: id, InstructorID: instructorID}, nil
}

func (s *Store) Validate(ctx context.Context, cookieValue string, now time.Time) (SessionData, error) {
	s.init()
	var data SessionData
	var expiresAt, absoluteExpiresAt string
	err := s.DB.QueryRowContext(ctx, "SELECT id, instructor_id, expires_at, absolute_expires_at FROM sessions WHERE id = ?", cookieValue).Scan(&data.ID, &data.InstructorID, &expiresAt, &absoluteExpiresAt)
	if err != nil {
		return SessionData{}, ErrInvalidSession
	}
	exp, _ := time.Parse(time.RFC3339, expiresAt)
	abs, _ := time.Parse(time.RFC3339, absoluteExpiresAt)
	if now.UTC().After(exp) || now.UTC().After(abs) {
		_, _ = s.DB.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", cookieValue)
		return SessionData{}, ErrInvalidSession
	}
	_, _ = s.DB.ExecContext(ctx, "UPDATE sessions SET last_seen_at = ?, expires_at = ? WHERE id = ?", now.UTC().Format(time.RFC3339), now.UTC().Add(12*time.Hour).Format(time.RFC3339), cookieValue)
	return data, nil
}

func (s *Store) Destroy(ctx context.Context, cookieValue string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM sessions WHERE id = ?", cookieValue)
	return err
}
