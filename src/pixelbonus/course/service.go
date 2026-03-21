package course

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidCourse = errors.New("invalid course")
	ErrNotFound      = errors.New("not found")
)

var validMethods = map[string]bool{
	"zero_to_q2_10": true,
	"zero_to_q3_10": true,
	"zero_to_q4_10": true,
	"q2_5_to_q3_10": true,
	"q2_5_to_q4_10": true,
	"q1_5_to_q2_10": true,
	"q1_5_to_q3_10": true,
	"q1_5_to_q4_10": true,
}

type Service struct {
	db    *sql.DB
	clock func() time.Time
}

type Course struct {
	ID            int64
	InstructorID  int64
	Name          string
	GradingMethod string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CreateCourseInput struct {
	Name          string
	GradingMethod string
}

type UpdateCourseInput struct {
	Name          string
	GradingMethod string
}

type StudentRedemption struct {
	StudentID   string
	Redemptions int
}

func NewService(db *sql.DB, clock func() time.Time) *Service {
	return &Service{db: db, clock: clock}
}

func (s *Service) ListCourses(ctx context.Context, instructorID int64) ([]Course, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, instructor_id, name, grading_method, created_at, updated_at
		FROM courses
		WHERE instructor_id = ?
		ORDER BY created_at DESC, id DESC
	`, instructorID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var courses []Course
	for rows.Next() {
		var c Course
		if err := rows.Scan(&c.ID, &c.InstructorID, &c.Name, &c.GradingMethod, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		courses = append(courses, c)
	}
	return courses, rows.Err()
}

func (s *Service) CreateCourse(ctx context.Context, instructorID int64, input CreateCourseInput) (Course, error) {
	if err := validate(input.Name, input.GradingMethod); err != nil {
		return Course{}, err
	}
	now := s.clock().UTC()
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO courses (instructor_id, name, grading_method, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, instructorID, strings.TrimSpace(input.Name), input.GradingMethod, now, now)
	if err != nil {
		return Course{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Course{}, err
	}
	return s.GetCourse(ctx, instructorID, id)
}

func (s *Service) GetCourse(ctx context.Context, instructorID int64, courseID int64) (Course, error) {
	var c Course
	err := s.db.QueryRowContext(ctx, `
		SELECT id, instructor_id, name, grading_method, created_at, updated_at
		FROM courses
		WHERE id = ? AND instructor_id = ?
	`, courseID, instructorID).Scan(&c.ID, &c.InstructorID, &c.Name, &c.GradingMethod, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return Course{}, ErrNotFound
	}
	if err != nil {
		return Course{}, err
	}
	return c, nil
}

func (s *Service) UpdateCourse(ctx context.Context, instructorID int64, courseID int64, input UpdateCourseInput) (Course, error) {
	if err := validate(input.Name, input.GradingMethod); err != nil {
		return Course{}, err
	}
	res, err := s.db.ExecContext(ctx, `
		UPDATE courses
		SET name = ?, grading_method = ?, updated_at = ?
		WHERE id = ? AND instructor_id = ?
	`, strings.TrimSpace(input.Name), input.GradingMethod, s.clock().UTC(), courseID, instructorID)
	if err != nil {
		return Course{}, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return Course{}, err
	}
	if affected == 0 {
		return Course{}, ErrNotFound
	}
	return s.GetCourse(ctx, instructorID, courseID)
}

func (s *Service) PublicRedemptions(ctx context.Context, courseID int64) (Course, []StudentRedemption, error) {
	var c Course
	err := s.db.QueryRowContext(ctx, `
		SELECT id, instructor_id, name, grading_method, created_at, updated_at
		FROM courses
		WHERE id = ?
	`, courseID).Scan(&c.ID, &c.InstructorID, &c.Name, &c.GradingMethod, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return Course{}, nil, ErrNotFound
	}
	if err != nil {
		return Course{}, nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT redeemed_by_student_id, COUNT(*)
		FROM coupons
		WHERE course_id = ? AND redeemed_by_student_id IS NOT NULL
		GROUP BY redeemed_by_student_id
		ORDER BY redeemed_by_student_id ASC
	`, courseID)
	if err != nil {
		return Course{}, nil, err
	}
	defer rows.Close()

	var result []StudentRedemption
	for rows.Next() {
		var row StudentRedemption
		if err := rows.Scan(&row.StudentID, &row.Redemptions); err != nil {
			return Course{}, nil, err
		}
		result = append(result, row)
	}
	return c, result, rows.Err()
}

func validate(name string, gradingMethod string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: course name is required", ErrInvalidCourse)
	}
	if !validMethods[gradingMethod] {
		return fmt.Errorf("%w: grading method is invalid", ErrInvalidCourse)
	}
	return nil
}
