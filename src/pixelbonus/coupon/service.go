package coupon

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

var (
	ErrInvalidCoupon = errors.New("invalid coupon")
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotFound      = errors.New("not found")
	ErrNoJobs        = errors.New("no jobs")
)

type Service struct {
	db     *sql.DB
	clock  func() time.Time
	random io.Reader
}

type GenerationJob struct {
	ID              int64
	InstructorID    int64
	InstructorEmail string
	CourseID        int64
	CourseName      string
	Quantity        int
	Status          string
	CreatedAt       time.Time
}

type CouponToken struct {
	Raw        []byte
	Normalized string
	Grouped    string
}

type StoredFile struct {
	ID           int64
	JobID        int64
	Nonce        string
	AbsolutePath string
	CreatedAt    time.Time
}

type RedeemResult struct {
	Status        string
	CourseName    string
	CouponCode    string
	StudentID     string
	AlreadyRedeem bool
}

type CouponLookup struct {
	CourseID   int64
	CourseName string
	CouponCode string
}

func NewService(db *sql.DB, clock func() time.Time, random io.Reader) *Service {
	return &Service{db: db, clock: clock, random: random}
}

func (s *Service) QueueGeneration(ctx context.Context, instructorID int64, courseID int64, quantity int) (GenerationJob, error) {
	if quantity <= 0 || quantity%20 != 0 {
		return GenerationJob{}, ErrInvalidInput
	}
	var courseName string
	err := s.db.QueryRowContext(ctx, `SELECT name FROM courses WHERE id = ? AND instructor_id = ?`, courseID, instructorID).Scan(&courseName)
	if err == sql.ErrNoRows {
		return GenerationJob{}, ErrNotFound
	}
	if err != nil {
		return GenerationJob{}, err
	}
	now := s.clock().UTC()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO coupon_generation_jobs (instructor_id, course_id, quantity, status, error_message, created_at, started_at, completed_at)
		VALUES (?, ?, ?, 'queued', NULL, ?, NULL, NULL)
	`, instructorID, courseID, quantity, now)
	if err != nil {
		return GenerationJob{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return GenerationJob{}, err
	}
	return GenerationJob{ID: id, InstructorID: instructorID, CourseID: courseID, CourseName: courseName, Quantity: quantity, Status: "queued", CreatedAt: now}, nil
}

func (s *Service) RedeemByCode(ctx context.Context, code string, studentID string) (RedeemResult, error) {
	normalized, err := NormalizeCode(code)
	if err != nil {
		return RedeemResult{}, err
	}
	studentID = strings.TrimSpace(studentID)
	if studentID == "" || len([]rune(studentID)) > 20 {
		return RedeemResult{}, ErrInvalidInput
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RedeemResult{}, err
	}
	defer tx.Rollback()

	var couponID int64
	var courseName string
	var redeemedBy sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT c.id, courses.name, c.redeemed_by_student_id
		FROM coupons c
		JOIN courses ON courses.id = c.course_id
		WHERE c.normalized_code = ?
	`, normalized).Scan(&couponID, &courseName, &redeemedBy)
	if err == sql.ErrNoRows {
		return RedeemResult{Status: "invalid", CouponCode: grouped(normalized)}, ErrInvalidCoupon
	}
	if err != nil {
		return RedeemResult{}, err
	}
	if redeemedBy.Valid {
		return RedeemResult{Status: "already_redeemed", CourseName: courseName, CouponCode: grouped(normalized)}, nil
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE coupons
		SET redeemed_by_student_id = ?, redeemed_at = ?
		WHERE id = ? AND redeemed_by_student_id IS NULL
	`, studentID, s.clock().UTC(), couponID)
	if err != nil {
		return RedeemResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return RedeemResult{}, err
	}
	return RedeemResult{Status: "redeemed", CourseName: courseName, CouponCode: grouped(normalized), StudentID: studentID}, nil
}

func (s *Service) LookupCoupon(ctx context.Context, code string) (CouponLookup, error) {
	normalized, err := NormalizeCode(code)
	if err != nil {
		return CouponLookup{}, err
	}
	var result CouponLookup
	err = s.db.QueryRowContext(ctx, `
		SELECT c.course_id, courses.name, c.normalized_code
		FROM coupons c
		JOIN courses ON courses.id = c.course_id
		WHERE c.normalized_code = ?
	`, normalized).Scan(&result.CourseID, &result.CourseName, &result.CouponCode)
	if err == sql.ErrNoRows {
		return CouponLookup{}, ErrInvalidCoupon
	}
	if err != nil {
		return CouponLookup{}, err
	}
	result.CouponCode = grouped(result.CouponCode)
	return result, nil
}

func (s *Service) ClaimNextJob(ctx context.Context) (GenerationJob, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return GenerationJob{}, err
	}
	defer tx.Rollback()

	var job GenerationJob
	err = tx.QueryRowContext(ctx, `
		SELECT j.id, j.instructor_id, i.email, j.course_id, c.name, j.quantity, j.status, j.created_at
		FROM coupon_generation_jobs j
		JOIN instructors i ON i.id = j.instructor_id
		JOIN courses c ON c.id = j.course_id
		WHERE j.status = 'queued'
		ORDER BY j.created_at ASC, j.id ASC
		LIMIT 1
	`).Scan(&job.ID, &job.InstructorID, &job.InstructorEmail, &job.CourseID, &job.CourseName, &job.Quantity, &job.Status, &job.CreatedAt)
	if err == sql.ErrNoRows {
		return GenerationJob{}, ErrNoJobs
	}
	if err != nil {
		return GenerationJob{}, err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE coupon_generation_jobs SET status = 'running', started_at = ? WHERE id = ?`, s.clock().UTC(), job.ID); err != nil {
		return GenerationJob{}, err
	}
	job.Status = "running"
	if err := tx.Commit(); err != nil {
		return GenerationJob{}, err
	}
	return job, nil
}

func (s *Service) CompleteJob(ctx context.Context, jobID int64, tokens []CouponToken, file StoredFile) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var courseID int64
	if err := tx.QueryRowContext(ctx, `SELECT course_id FROM coupon_generation_jobs WHERE id = ?`, jobID).Scan(&courseID); err != nil {
		return err
	}
	now := s.clock().UTC()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO coupon_sets (course_id, job_id, created_at)
		VALUES (?, ?, ?)
	`, courseID, jobID, now)
	if err != nil {
		return err
	}
	setID, err := res.LastInsertId()
	if err != nil {
		return err
	}
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO coupons (coupon_set_id, course_id, raw_code, normalized_code, redeemed_by_student_id, redeemed_at)
		VALUES (?, ?, ?, ?, NULL, NULL)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, token := range tokens {
		if _, err := stmt.ExecContext(ctx, setID, courseID, token.Raw, token.Normalized); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO generated_files (job_id, nonce, absolute_path, created_at)
		VALUES (?, ?, ?, ?)
	`, jobID, file.Nonce, file.AbsolutePath, now); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE coupon_generation_jobs
		SET status = 'completed', completed_at = ?, error_message = NULL
		WHERE id = ?
	`, now, jobID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) FailJob(ctx context.Context, jobID int64, message string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE coupon_generation_jobs
		SET status = 'failed', completed_at = ?, error_message = ?
		WHERE id = ?
	`, s.clock().UTC(), message, jobID)
	return err
}

func (s *Service) FileByNonce(ctx context.Context, nonce string) (StoredFile, error) {
	var file StoredFile
	err := s.db.QueryRowContext(ctx, `
		SELECT id, job_id, nonce, absolute_path, created_at
		FROM generated_files
		WHERE nonce = ?
	`, nonce).Scan(&file.ID, &file.JobID, &file.Nonce, &file.AbsolutePath, &file.CreatedAt)
	if err == sql.ErrNoRows {
		return StoredFile{}, ErrNotFound
	}
	if err != nil {
		return StoredFile{}, err
	}
	return file, nil
}

func NormalizeCode(value string) (string, error) {
	compact := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
	if len(compact) != 40 {
		return "", ErrInvalidCoupon
	}
	for _, r := range compact {
		if !(r >= '0' && r <= '9' || r >= 'A' && r <= 'F') {
			return "", ErrInvalidCoupon
		}
	}
	return compact, nil
}

func GenerateTokens(random io.Reader, quantity int) ([]CouponToken, error) {
	tokens := make([]CouponToken, 0, quantity)
	for i := 0; i < quantity; i++ {
		raw := make([]byte, 20)
		if _, err := io.ReadFull(random, raw); err != nil {
			return nil, fmt.Errorf("read random: %w", err)
		}
		normalized := strings.ToUpper(hex.EncodeToString(raw))
		tokens = append(tokens, CouponToken{Raw: raw, Normalized: normalized, Grouped: grouped(normalized)})
	}
	return tokens, nil
}

func GenerateFileNonce(random io.Reader) (string, error) {
	buf := make([]byte, 16)
	if _, err := io.ReadFull(random, buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func grouped(normalized string) string {
	var parts []string
	for i := 0; i < len(normalized); i += 4 {
		parts = append(parts, normalized[i:i+4])
	}
	return strings.Join(parts, " ")
}
